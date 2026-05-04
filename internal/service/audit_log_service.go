package service

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/Zyx-98/auth-service/internal/domain/entity"
	"github.com/Zyx-98/auth-service/internal/domain/repository"
	"github.com/Zyx-98/auth-service/internal/dto"
)

type AuditLogService struct {
	auditLogRepo repository.AuditLogRepository
}

func NewAuditLogService(auditLogRepo repository.AuditLogRepository) *AuditLogService {
	return &AuditLogService{
		auditLogRepo: auditLogRepo,
	}
}

func (s *AuditLogService) LogAuthEvent(ctx context.Context, actorID *uuid.UUID, eventType, action, status string, metadata map[string]any) error {
	var metadataJSON *string
	if metadata != nil {
		data, err := json.Marshal(metadata)
		if err != nil {
			log.Printf("Failed to marshal audit metadata: %v", err)
			return nil
		}
		str := string(data)
		metadataJSON = &str
	}

	auditLog := entity.NewAuditLog(eventType, action, status, actorID)
	auditLog.Metadata = metadataJSON

	if err := s.auditLogRepo.Create(ctx, auditLog); err != nil {
		log.Printf("Failed to create audit log: %v", err)
		return nil
	}

	return nil
}

func (s *AuditLogService) LogRBACEvent(ctx context.Context, actorID, resourceID uuid.UUID, action string, oldValue, newValue any, status string) error {
	var oldValueJSON, newValueJSON *string

	if oldValue != nil {
		data, err := json.Marshal(oldValue)
		if err != nil {
			log.Printf("Failed to marshal old value: %v", err)
			return nil
		}
		str := string(data)
		oldValueJSON = &str
	}

	if newValue != nil {
		data, err := json.Marshal(newValue)
		if err != nil {
			log.Printf("Failed to marshal new value: %v", err)
			return nil
		}
		str := string(data)
		newValueJSON = &str
	}

	resourceType := "role"
	auditLog := entity.NewAuditLog("role.assignment", action, status, &actorID)
	auditLog.ResourceID = &resourceID
	auditLog.ResourceType = &resourceType
	auditLog.OldValue = oldValueJSON
	auditLog.NewValue = newValueJSON

	if err := s.auditLogRepo.Create(ctx, auditLog); err != nil {
		log.Printf("Failed to create RBAC audit log: %v", err)
		return nil
	}

	return nil
}

func (s *AuditLogService) GetMyAuditLogs(ctx context.Context, actorID uuid.UUID, limit, offset int) ([]*dto.AuditLogResponse, error) {
	logs, err := s.auditLogRepo.ListByActorID(ctx, actorID, limit, offset)
	if err != nil {
		return nil, err
	}

	return s.entitiesToDTOs(logs), nil
}

func (s *AuditLogService) GetAuditLogs(ctx context.Context, filters repository.AuditLogFilters, limit, offset int) ([]*dto.AuditLogResponse, error) {
	logs, err := s.auditLogRepo.List(ctx, filters, limit, offset)
	if err != nil {
		return nil, err
	}

	return s.entitiesToDTOs(logs), nil
}

func (s *AuditLogService) entitiesToDTOs(logs []*entity.AuditLog) []*dto.AuditLogResponse {
	responses := make([]*dto.AuditLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = &dto.AuditLogResponse{
			ID:           log.ID,
			EventType:    log.EventType,
			ActorID:      log.ActorID,
			ResourceID:   log.ResourceID,
			ResourceType: log.ResourceType,
			Action:       log.Action,
			Status:       log.Status,
			StatusReason: log.StatusReason,
			Metadata:     log.Metadata,
			CreatedAt:    log.CreatedAt,
		}
	}
	return responses
}
