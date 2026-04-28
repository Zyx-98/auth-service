package dto

import (
	"time"

	"github.com/google/uuid"
)

type AuditLogResponse struct {
	ID           uuid.UUID  `json:"id"`
	EventType    string     `json:"event_type"`
	ActorID      *uuid.UUID `json:"actor_id,omitempty"`
	ResourceID   *uuid.UUID `json:"resource_id,omitempty"`
	ResourceType *string    `json:"resource_type,omitempty"`
	Action       string     `json:"action"`
	Status       string     `json:"status"`
	StatusReason *string    `json:"status_reason,omitempty"`
	Metadata     *string    `json:"metadata,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type ListAuditLogsRequest struct {
	Limit      int        `form:"limit"`
	Offset     int        `form:"offset"`
	ActorID    *uuid.UUID `form:"actor_id"`
	ResourceID *uuid.UUID `form:"resource_id"`
	EventType  *string    `form:"event_type"`
}
