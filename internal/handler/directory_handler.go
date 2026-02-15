package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/psds-microservice/operator-directory-service/internal/errs"
	"github.com/psds-microservice/operator-directory-service/internal/model"
	"github.com/psds-microservice/operator-directory-service/internal/service"
	"github.com/psds-microservice/operator-directory-service/internal/validator"
)

type DirectoryHandler struct {
	svc *service.DirectoryService
}

func NewDirectoryHandler(svc *service.DirectoryService) *DirectoryHandler {
	return &DirectoryHandler{svc: svc}
}

// List GET /operators?region=&role=&status=&limit=20&offset=0
func (h *DirectoryHandler) List(c *gin.Context) {
	region := c.Query("region")
	role := c.Query("role")
	status := c.Query("status")
	limit, _ := parseIntDefault(c.Query("limit"), 20)
	offset, _ := parseIntDefault(c.Query("offset"), 0)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	result, err := h.svc.List(c.Request.Context(), region, role, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func parseIntDefault(s string, def int) (int, bool) {
	if s == "" {
		return def, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def, false
	}
	return n, true
}

// Get GET /operators/:id
func (h *DirectoryHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	entry, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, errs.ErrOperatorNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "operator not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entry)
}

// Create POST /operators (body: user_id, region?, role?, display_name?)
func (h *DirectoryHandler) Create(c *gin.Context) {
	var req struct {
		UserID      string `json:"user_id" binding:"required"`
		Region      string `json:"region"`
		Role        string `json:"role"`
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	role := req.Role
	if role == "" {
		role = "operator"
	}
	if !validator.IsValidRole(role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role: must be one of operator, supervisor, admin"})
		return
	}
	p := &model.OperatorProfile{
		UserID:      userID,
		Region:      req.Region,
		Role:        role,
		DisplayName: req.DisplayName,
	}
	if err := h.svc.Create(c.Request.Context(), p); err != nil {
		if errors.Is(err, errs.ErrOperatorAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "operator profile already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

// Update PUT /operators/:id (body: region?, role?, display_name? — only provided fields are updated)
func (h *DirectoryHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Region      *string `json:"region"`
		Role        *string `json:"role"`
		DisplayName *string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	p, err := h.svc.GetProfile(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, errs.ErrOperatorNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "operator not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if req.Region != nil {
		p.Region = *req.Region
	}
	if req.Role != nil && *req.Role != "" {
		if !validator.IsValidRole(*req.Role) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role: must be one of operator, supervisor, admin"})
			return
		}
		p.Role = *req.Role
	}
	if req.DisplayName != nil {
		p.DisplayName = *req.DisplayName
	}
	if err := h.svc.Update(c.Request.Context(), p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}
