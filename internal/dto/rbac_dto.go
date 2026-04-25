package dto

import "github.com/google/uuid"

type CreateRoleRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=100"`
	Description *string `json:"description"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=100"`
	Description *string `json:"description"`
}

type RoleResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
}

type CreatePermissionRequest struct {
	Name     string `json:"name" validate:"required,min=3,max=100"`
	Resource string `json:"resource" validate:"required,min=2,max=100"`
	Action   string `json:"action" validate:"required,min=2,max=50"`
}

type PermissionResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Resource string    `json:"resource"`
	Action   string    `json:"action"`
}

type AssignRoleRequest struct {
	RoleID uuid.UUID `json:"role_id" validate:"required"`
}

type AssignRoleResponse struct {
	UserID uuid.UUID `json:"user_id"`
	RoleID uuid.UUID `json:"role_id"`
}

type RemoveRoleRequest struct {
	RoleID uuid.UUID `json:"role_id" validate:"required"`
}

type RoleDetailResponse struct {
	ID          uuid.UUID             `json:"id"`
	Name        string                `json:"name"`
	Description *string               `json:"description"`
	Permissions []PermissionResponse   `json:"permissions"`
}

type UserRolesResponse struct {
	UserID      uuid.UUID             `json:"user_id"`
	Email       string                `json:"email"`
	Roles       []RoleResponse        `json:"roles"`
	Permissions []PermissionResponse  `json:"permissions"`
}
