package grpc

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/psds-microservice/operator-directory-service/internal/errs"
	"github.com/psds-microservice/operator-directory-service/internal/model"
	"github.com/psds-microservice/operator-directory-service/internal/service"
	"github.com/psds-microservice/operator-directory-service/internal/validator"
	"github.com/psds-microservice/operator-directory-service/pkg/gen/operator_directory_service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Deps — зависимости gRPC-сервера (D: зависимость от абстракций).
type Deps struct {
	Directory service.DirectoryServicer
}

// Server implements operator_directory_service.OperatorDirectoryServiceServer
type Server struct {
	operator_directory_service.UnimplementedOperatorDirectoryServiceServer
	Deps
}

// NewServer создаёт gRPC-сервер с внедрёнными сервисами
func NewServer(deps Deps) *Server {
	return &Server{Deps: deps}
}

func (s *Server) mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errs.ErrOperatorNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, errs.ErrOperatorAlreadyExists) {
		return status.Error(codes.AlreadyExists, err.Error())
	}
	log.Printf("grpc: unhandled error: %v", err)
	return status.Error(codes.Internal, err.Error())
}

func toProtoOperatorEntry(entry *service.OperatorEntry) *operator_directory_service.OperatorEntry {
	if entry == nil {
		return nil
	}
	return &operator_directory_service.OperatorEntry{
		UserId:         entry.UserID,
		Region:         entry.Region,
		Role:           entry.Role,
		DisplayName:    entry.DisplayName,
		Available:      entry.Available,
		ActiveSessions: int32(entry.ActiveSessions),
		MaxSessions:    int32(entry.MaxSessions),
	}
}

func toProtoOperatorProfile(profile *model.OperatorProfile) *operator_directory_service.OperatorProfile {
	if profile == nil {
		return nil
	}
	out := &operator_directory_service.OperatorProfile{
		UserId:      profile.UserID.String(),
		Region:      profile.Region,
		Role:        profile.Role,
		DisplayName: profile.DisplayName,
	}
	if !profile.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(profile.CreatedAt)
	}
	if !profile.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(profile.UpdatedAt)
	}
	return out
}

func (s *Server) ListOperators(ctx context.Context, req *operator_directory_service.ListOperatorsRequest) (*operator_directory_service.ListOperatorsResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := int(req.GetOffset())
	if offset < 0 {
		offset = 0
	}

	result, err := s.Directory.List(ctx, req.GetRegion(), req.GetRole(), req.GetStatus(), limit, offset)
	if err != nil {
		return nil, s.mapError(err)
	}

	operators := make([]*operator_directory_service.OperatorEntry, len(result.Operators))
	for i, entry := range result.Operators {
		operators[i] = toProtoOperatorEntry(&entry)
	}

	return &operator_directory_service.ListOperatorsResponse{
		Operators: operators,
		Total:     int32(result.Total),
		Limit:     int32(result.Limit),
		Offset:    int32(result.Offset),
	}, nil
}

func (s *Server) GetOperator(ctx context.Context, req *operator_directory_service.GetOperatorRequest) (*operator_directory_service.OperatorEntry, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	entry, err := s.Directory.GetByID(ctx, userID)
	if err != nil {
		return nil, s.mapError(err)
	}
	return toProtoOperatorEntry(entry), nil
}

func (s *Server) CreateOperator(ctx context.Context, req *operator_directory_service.CreateOperatorRequest) (*operator_directory_service.OperatorProfile, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	role := req.GetRole()
	if role == "" {
		role = "operator"
	}
	if !validator.IsValidRole(role) {
		return nil, status.Error(codes.InvalidArgument, "invalid role: must be one of operator, supervisor, admin")
	}

	profile := &model.OperatorProfile{
		UserID:      userID,
		Region:      req.GetRegion(),
		Role:        role,
		DisplayName: req.GetDisplayName(),
	}

	if err := s.Directory.Create(ctx, profile); err != nil {
		return nil, s.mapError(err)
	}

	return toProtoOperatorProfile(profile), nil
}

func (s *Server) UpdateOperator(ctx context.Context, req *operator_directory_service.UpdateOperatorRequest) (*operator_directory_service.OperatorProfile, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	profile, err := s.Directory.GetProfile(ctx, userID)
	if err != nil {
		return nil, s.mapError(err)
	}

	if req.GetRegion() != "" {
		profile.Region = req.GetRegion()
	}
	if req.GetRole() != "" {
		if !validator.IsValidRole(req.GetRole()) {
			return nil, status.Error(codes.InvalidArgument, "invalid role: must be one of operator, supervisor, admin")
		}
		profile.Role = req.GetRole()
	}
	if req.GetDisplayName() != "" {
		profile.DisplayName = req.GetDisplayName()
	}

	if err := s.Directory.Update(ctx, profile); err != nil {
		return nil, s.mapError(err)
	}

	return toProtoOperatorProfile(profile), nil
}
