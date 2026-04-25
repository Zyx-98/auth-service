package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/dto"
	"github.com/hatuan/auth-service/internal/service"
	"github.com/hatuan/auth-service/pkg/apperror"
	"github.com/hatuan/auth-service/pkg/response"
	"github.com/hatuan/auth-service/pkg/validator"
)

type RBACHandler struct {
	rbacService *service.RBACService
}

func NewRBACHandler(rbacService *service.RBACService) *RBACHandler {
	return &RBACHandler{
		rbacService: rbacService,
	}
}

// Role Handlers

func (h *RBACHandler) CreateRole(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid request", err))
		return
	}

	validationErrs := validator.Validate(req)
	if len(validationErrs) > 0 {
		response.ValidationErrors(c, validationErrs)
		return
	}

	roleResp, err := h.rbacService.CreateRole(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, roleResp)
}

func (h *RBACHandler) GetRole(c *gin.Context) {
	roleID := c.Param("id")
	id, err := uuid.Parse(roleID)
	if err != nil {
		response.Error(c, apperror.BadRequest("Invalid role ID", err))
		return
	}

	roleResp, err := h.rbacService.GetRole(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Ok(c, roleResp)
}

func (h *RBACHandler) ListRoles(c *gin.Context) {
	roles, err := h.rbacService.ListRoles(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Ok(c, gin.H{"roles": roles})
}

func (h *RBACHandler) UpdateRole(c *gin.Context) {
	roleID := c.Param("id")
	id, err := uuid.Parse(roleID)
	if err != nil {
		response.Error(c, apperror.BadRequest("Invalid role ID", err))
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid request", err))
		return
	}

	validationErrs := validator.Validate(req)
	if len(validationErrs) > 0 {
		response.ValidationErrors(c, validationErrs)
		return
	}

	roleResp, err := h.rbacService.UpdateRole(c.Request.Context(), id, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Ok(c, roleResp)
}

func (h *RBACHandler) DeleteRole(c *gin.Context) {
	roleID := c.Param("id")
	id, err := uuid.Parse(roleID)
	if err != nil {
		response.Error(c, apperror.BadRequest("Invalid role ID", err))
		return
	}

	if err := h.rbacService.DeleteRole(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	response.NoContent(c)
}

// Permission Handlers

func (h *RBACHandler) CreatePermission(c *gin.Context) {
	var req dto.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid request", err))
		return
	}

	validationErrs := validator.Validate(req)
	if len(validationErrs) > 0 {
		response.ValidationErrors(c, validationErrs)
		return
	}

	permResp, err := h.rbacService.CreatePermission(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, permResp)
}

func (h *RBACHandler) ListPermissions(c *gin.Context) {
	permissions, err := h.rbacService.ListPermissions(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Ok(c, gin.H{"permissions": permissions})
}

func (h *RBACHandler) DeletePermission(c *gin.Context) {
	permID := c.Param("id")
	id, err := uuid.Parse(permID)
	if err != nil {
		response.Error(c, apperror.BadRequest("Invalid permission ID", err))
		return
	}

	if err := h.rbacService.DeletePermission(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	response.NoContent(c)
}

// User Role Assignment Handlers

func (h *RBACHandler) AssignRoleToUser(c *gin.Context) {
	userID := c.Param("user_id")
	uID, err := uuid.Parse(userID)
	if err != nil {
		response.Error(c, apperror.BadRequest("Invalid user ID", err))
		return
	}

	var req dto.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest("Invalid request", err))
		return
	}

	validationErrs := validator.Validate(req)
	if len(validationErrs) > 0 {
		response.ValidationErrors(c, validationErrs)
		return
	}

	assignResp, err := h.rbacService.AssignRoleToUser(c.Request.Context(), uID, req.RoleID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, assignResp)
}

func (h *RBACHandler) RemoveRoleFromUser(c *gin.Context) {
	userID := c.Param("user_id")
	uID, err := uuid.Parse(userID)
	if err != nil {
		response.Error(c, apperror.BadRequest("Invalid user ID", err))
		return
	}

	roleID := c.Param("role_id")
	rID, err := uuid.Parse(roleID)
	if err != nil {
		response.Error(c, apperror.BadRequest("Invalid role ID", err))
		return
	}

	if err := h.rbacService.RemoveRoleFromUser(c.Request.Context(), uID, rID); err != nil {
		response.Error(c, err)
		return
	}

	response.NoContent(c)
}

func (h *RBACHandler) GetUserRoles(c *gin.Context) {
	userID := c.Param("user_id")
	uID, err := uuid.Parse(userID)
	if err != nil {
		response.Error(c, apperror.BadRequest("Invalid user ID", err))
		return
	}

	userRoles, err := h.rbacService.GetUserRoles(c.Request.Context(), uID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Ok(c, userRoles)
}
