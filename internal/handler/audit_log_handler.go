package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/Zyx-98/auth-service/internal/domain/repository"
	"github.com/Zyx-98/auth-service/internal/service"
	"github.com/Zyx-98/auth-service/pkg/apperror"
	"github.com/Zyx-98/auth-service/pkg/response"
)

type AuditLogHandler struct {
	auditLogService *service.AuditLogService
}

func NewAuditLogHandler(auditLogService *service.AuditLogService) *AuditLogHandler {
	return &AuditLogHandler{
		auditLogService: auditLogService,
	}
}

func (h *AuditLogHandler) GetMyAuditLogs(c *gin.Context) {
	var req struct {
		Limit  int `form:"limit,default=50"`
		Offset int `form:"offset,default=0"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid query parameters", err))
		return
	}

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperror.Unauthorized("Missing user ID"))
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Error(c, apperror.InternalServerError("Invalid user ID type", nil))
		return
	}

	logs, err := h.auditLogService.GetMyAuditLogs(c.Request.Context(), userID, req.Limit, req.Offset)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Ok(c, gin.H{
		"data": logs,
	})
}

func (h *AuditLogHandler) GetAuditLogs(c *gin.Context) {
	var req struct {
		Limit      int       `form:"limit,default=50"`
		Offset     int       `form:"offset,default=0"`
		ActorID    *uuid.UUID `form:"actor_id"`
		ResourceID *uuid.UUID `form:"resource_id"`
		EventType  *string   `form:"event_type"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid query parameters", err))
		return
	}

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	filters := repository.AuditLogFilters{
		ActorID:    req.ActorID,
		ResourceID: req.ResourceID,
		EventType:  req.EventType,
	}

	logs, err := h.auditLogService.GetAuditLogs(c.Request.Context(), filters, req.Limit, req.Offset)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Ok(c, gin.H{
		"data": logs,
	})
}
