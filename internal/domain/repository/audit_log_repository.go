package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/domain/entity"
)

type AuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
	ListByActorID(ctx context.Context, actorID uuid.UUID, limit, offset int) ([]*entity.AuditLog, error)
	List(ctx context.Context, filters AuditLogFilters, limit, offset int) ([]*entity.AuditLog, error)
}

type AuditLogFilters struct {
	ActorID      *uuid.UUID
	ResourceID   *uuid.UUID
	EventType    *string
}
