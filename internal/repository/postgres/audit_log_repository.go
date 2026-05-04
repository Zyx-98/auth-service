package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/Zyx-98/auth-service/internal/domain/entity"
	"github.com/Zyx-98/auth-service/internal/domain/repository"
	"gorm.io/gorm"
)

type auditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) repository.AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(ctx context.Context, log *entity.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *auditLogRepository) ListByActorID(ctx context.Context, actorID uuid.UUID, limit, offset int) ([]*entity.AuditLog, error) {
	var logs []*entity.AuditLog
	err := r.db.WithContext(ctx).
		Where("actor_id = ?", actorID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *auditLogRepository) List(ctx context.Context, filters repository.AuditLogFilters, limit, offset int) ([]*entity.AuditLog, error) {
	var logs []*entity.AuditLog
	query := r.db.WithContext(ctx)

	if filters.ActorID != nil {
		query = query.Where("actor_id = ?", *filters.ActorID)
	}
	if filters.ResourceID != nil {
		query = query.Where("resource_id = ?", *filters.ResourceID)
	}
	if filters.EventType != nil {
		query = query.Where("event_type = ?", *filters.EventType)
	}

	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}
