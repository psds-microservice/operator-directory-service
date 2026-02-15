package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/psds-microservice/operator-directory-service/internal/client"
	"github.com/psds-microservice/operator-directory-service/internal/errs"
	"github.com/psds-microservice/operator-directory-service/internal/model"
	"gorm.io/gorm"
)

type DirectoryService struct {
	db   *gorm.DB
	pool *client.PoolClient
}

func NewDirectoryService(db *gorm.DB, pool *client.PoolClient) *DirectoryService {
	return &DirectoryService{db: db, pool: pool}
}

type OperatorEntry struct {
	UserID         string `json:"user_id"`
	Region         string `json:"region,omitempty"`
	Role           string `json:"role"`
	DisplayName    string `json:"display_name,omitempty"`
	Available      bool   `json:"available"`
	ActiveSessions int    `json:"active_sessions"`
	MaxSessions    int    `json:"max_sessions"`
}

// ListResult — результат списка с пагинацией.
type ListResult struct {
	Operators []OperatorEntry `json:"operators"`
	Total     int             `json:"total"`
	Limit     int             `json:"limit"`
	Offset    int             `json:"offset"`
}

func (s *DirectoryService) List(ctx context.Context, region, role, status string, limit, offset int) (*ListResult, error) {
	poolList, err := s.pool.ListOperators(ctx)
	if err != nil {
		log.Printf("[operator-directory] operator-pool list failed: %v", err)
		poolList = nil
	}

	var profileIDs []uuid.UUID
	for _, op := range poolList {
		u, _ := uuid.Parse(op.UserID)
		profileIDs = append(profileIDs, u)
	}
	var profiles []model.OperatorProfile
	if len(profileIDs) > 0 {
		s.db.WithContext(ctx).Where("user_id IN ?", profileIDs).Find(&profiles)
	}
	profileByID := make(map[string]model.OperatorProfile)
	for _, p := range profiles {
		profileByID[p.UserID.String()] = p
	}

	var all []OperatorEntry
	for _, po := range poolList {
		if status == "available" && (!po.Available || po.ActiveSessions >= po.MaxSessions) {
			continue
		}
		if status == "busy" && (po.Available && po.ActiveSessions < po.MaxSessions) {
			continue
		}
		p := profileByID[po.UserID]
		if region != "" && p.Region != region {
			continue
		}
		if role != "" && p.Role != role {
			continue
		}
		entry := OperatorEntry{
			UserID:         po.UserID,
			Region:         p.Region,
			Role:           p.Role,
			DisplayName:    p.DisplayName,
			Available:      po.Available,
			ActiveSessions: po.ActiveSessions,
			MaxSessions:    po.MaxSessions,
		}
		if p.Role == "" {
			entry.Role = "operator"
		}
		all = append(all, entry)
	}
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	page := all[offset:end]
	return &ListResult{Operators: page, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *DirectoryService) GetByID(ctx context.Context, userID uuid.UUID) (*OperatorEntry, error) {
	var p model.OperatorProfile
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrOperatorNotFound
		}
		return nil, err
	}
	poolList, _ := s.pool.ListOperators(ctx)
	uid := p.UserID.String()
	entry := &OperatorEntry{
		UserID:      uid,
		Region:      p.Region,
		Role:        p.Role,
		DisplayName: p.DisplayName,
	}
	for _, op := range poolList {
		if op.UserID == uid {
			entry.Available = op.Available
			entry.ActiveSessions = op.ActiveSessions
			entry.MaxSessions = op.MaxSessions
			break
		}
	}
	return entry, nil
}

func (s *DirectoryService) Create(ctx context.Context, p *model.OperatorProfile) error {
	var existing model.OperatorProfile
	err := s.db.WithContext(ctx).Where("user_id = ?", p.UserID).First(&existing).Error
	if err == nil {
		return errs.ErrOperatorAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return s.db.WithContext(ctx).Create(p).Error
}

func (s *DirectoryService) Update(ctx context.Context, p *model.OperatorProfile) error {
	p.UpdatedAt = time.Now()
	return s.db.WithContext(ctx).Model(p).Updates(map[string]interface{}{
		"region":       p.Region,
		"role":         p.Role,
		"display_name": p.DisplayName,
		"updated_at":   p.UpdatedAt,
	}).Error
}

func (s *DirectoryService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.OperatorProfile, error) {
	var p model.OperatorProfile
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrOperatorNotFound
		}
		return nil, err
	}
	return &p, nil
}
