package entity

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	EventType    string     `gorm:"not null"`
	ActorID      *uuid.UUID `gorm:"type:uuid"`
	ResourceID   *uuid.UUID `gorm:"type:uuid"`
	ResourceType *string
	Action       string `gorm:"not null"`
	Status       string `gorm:"not null"`
	StatusReason *string
	OldValue     *string `gorm:"type:jsonb"`
	NewValue     *string `gorm:"type:jsonb"`
	Metadata     *string `gorm:"type:jsonb"`
	CreatedAt    time.Time
}

func (a *AuditLog) TableName() string {
	return "audit_logs"
}

func NewAuditLog(eventType, action, status string, actorID *uuid.UUID) *AuditLog {
	return &AuditLog{
		ID:        uuid.New(),
		EventType: eventType,
		ActorID:   actorID,
		Action:    action,
		Status:    status,
		CreatedAt: time.Now(),
	}
}
