package model

import (
	"time"

	"github.com/google/uuid"
)

type OperatorProfile struct {
	UserID      uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	Region      string    `gorm:"size:64;index" json:"region,omitempty"`
	Role        string    `gorm:"size:32;not null;default:operator" json:"role"`
	DisplayName string    `gorm:"size:255" json:"display_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (OperatorProfile) TableName() string { return "operator_profiles" }
