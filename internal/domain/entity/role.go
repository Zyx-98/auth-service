package entity

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID   `gorm:"type:uuid;primaryKey"`
	Name        string      `gorm:"uniqueIndex;not null"`
	Description *string
	CreatedAt   time.Time
	Permissions []Permission `gorm:"many2many:role_permissions"`
}

func (r *Role) TableName() string {
	return "roles"
}

func NewRole(name string, description *string) *Role {
	return &Role{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}
}
