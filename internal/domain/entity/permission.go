package entity

import "github.com/google/uuid"

type Permission struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name     string    `gorm:"uniqueIndex;not null"`
	Resource string    `gorm:"not null"`
	Action   string    `gorm:"not null"`
}

func (p *Permission) TableName() string {
	return "permissions"
}

func NewPermission(name, resource, action string) *Permission {
	return &Permission{
		ID:       uuid.New(),
		Name:     name,
		Resource: resource,
		Action:   action,
	}
}
