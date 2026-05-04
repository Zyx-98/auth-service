package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/Zyx-98/auth-service/internal/domain/entity"
	"github.com/Zyx-98/auth-service/internal/domain/repository"
	"github.com/Zyx-98/auth-service/internal/dto"
	"github.com/Zyx-98/auth-service/pkg/apperror"
)

type RBACService struct {
	roleRepo        repository.RoleRepository
	permissionRepo  repository.PermissionRepository
	userRepo        repository.UserRepository
	auditLogService *AuditLogService
}

func NewRBACService(
	roleRepo repository.RoleRepository,
	permissionRepo repository.PermissionRepository,
	userRepo repository.UserRepository,
	auditLogService *AuditLogService,
) *RBACService {
	return &RBACService{
		roleRepo:        roleRepo,
		permissionRepo:  permissionRepo,
		userRepo:        userRepo,
		auditLogService: auditLogService,
	}
}

// Role Management

func (s *RBACService) CreateRole(ctx context.Context, req *dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	existing, err := s.roleRepo.GetByName(ctx, req.Name)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to check role", err)
	}

	if existing != nil {
		return nil, apperror.Conflict("Role already exists", nil)
	}

	role := entity.NewRole(req.Name, req.Description)

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, apperror.InternalServerError("Failed to create role", err)
	}

	return &dto.RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
	}, nil
}

func (s *RBACService) GetRole(ctx context.Context, id uuid.UUID) (*dto.RoleDetailResponse, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch role", err)
	}

	if role == nil {
		return nil, apperror.NotFound("Role not found")
	}

	permissions := make([]dto.PermissionResponse, len(role.Permissions))
	for i, p := range role.Permissions {
		permissions[i] = dto.PermissionResponse{
			ID:       p.ID,
			Name:     p.Name,
			Resource: p.Resource,
			Action:   p.Action,
		}
	}

	return &dto.RoleDetailResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		Permissions: permissions,
	}, nil
}

func (s *RBACService) ListRoles(ctx context.Context) ([]dto.RoleResponse, error) {
	roles, err := s.roleRepo.List(ctx)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to list roles", err)
	}

	response := make([]dto.RoleResponse, len(roles))
	for i, role := range roles {
		response[i] = dto.RoleResponse{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
		}
	}

	return response, nil
}

func (s *RBACService) UpdateRole(ctx context.Context, id uuid.UUID, req *dto.UpdateRoleRequest) (*dto.RoleResponse, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch role", err)
	}

	if role == nil {
		return nil, apperror.NotFound("Role not found")
	}

	if req.Name != role.Name {
		existing, err := s.roleRepo.GetByName(ctx, req.Name)
		if err != nil {
			return nil, apperror.InternalServerError("Failed to check role", err)
		}
		if existing != nil {
			return nil, apperror.Conflict("Role name already in use", nil)
		}
	}

	role.Name = req.Name
	role.Description = req.Description

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, apperror.InternalServerError("Failed to update role", err)
	}

	return &dto.RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
	}, nil
}

func (s *RBACService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return apperror.InternalServerError("Failed to fetch role", err)
	}

	if role == nil {
		return apperror.NotFound("Role not found")
	}

	if role.Name == "admin" || role.Name == "user" {
		return apperror.Conflict("Cannot delete system roles", nil)
	}

	if err := s.roleRepo.Delete(ctx, id); err != nil {
		return apperror.InternalServerError("Failed to delete role", err)
	}

	return nil
}

// Permission Management

func (s *RBACService) CreatePermission(ctx context.Context, req *dto.CreatePermissionRequest) (*dto.PermissionResponse, error) {
	existing, err := s.permissionRepo.GetByName(ctx, req.Name)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to check permission", err)
	}

	if existing != nil {
		return nil, apperror.Conflict("Permission already exists", nil)
	}

	permission := entity.NewPermission(req.Name, req.Resource, req.Action)

	if err := s.permissionRepo.Create(ctx, permission); err != nil {
		return nil, apperror.InternalServerError("Failed to create permission", err)
	}

	return &dto.PermissionResponse{
		ID:       permission.ID,
		Name:     permission.Name,
		Resource: permission.Resource,
		Action:   permission.Action,
	}, nil
}

func (s *RBACService) ListPermissions(ctx context.Context) ([]dto.PermissionResponse, error) {
	permissions, err := s.permissionRepo.List(ctx)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to list permissions", err)
	}

	response := make([]dto.PermissionResponse, len(permissions))
	for i, perm := range permissions {
		response[i] = dto.PermissionResponse{
			ID:       perm.ID,
			Name:     perm.Name,
			Resource: perm.Resource,
			Action:   perm.Action,
		}
	}

	return response, nil
}

func (s *RBACService) DeletePermission(ctx context.Context, id uuid.UUID) error {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		return apperror.InternalServerError("Failed to fetch permission", err)
	}

	if permission == nil {
		return apperror.NotFound("Permission not found")
	}

	if err := s.permissionRepo.Delete(ctx, id); err != nil {
		return apperror.InternalServerError("Failed to delete permission", err)
	}

	return nil
}

// User Role Assignment

func (s *RBACService) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) (*dto.AssignRoleResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return nil, apperror.NotFound("User not found")
	}

	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch role", err)
	}

	if role == nil {
		return nil, apperror.NotFound("Role not found")
	}

	if err := s.roleRepo.AssignRoleToUser(ctx, userID, roleID); err != nil {
		return nil, apperror.InternalServerError("Failed to assign role", err)
	}

	if err := s.auditLogService.LogRBACEvent(ctx, userID, roleID, "assign", map[string]string{"role": role.Name}, nil, "success"); err != nil {
		return nil, apperror.InternalServerError("Failed to log audit event", err)
	}

	return &dto.AssignRoleResponse{
		UserID: userID,
		RoleID: roleID,
	}, nil
}

func (s *RBACService) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return apperror.NotFound("User not found")
	}

	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return apperror.InternalServerError("Failed to fetch role", err)
	}

	if role == nil {
		return apperror.NotFound("Role not found")
	}

	if err := s.roleRepo.RemoveRoleFromUser(ctx, userID, roleID); err != nil {
		return apperror.InternalServerError("Failed to remove role", err)
	}

	if err := s.auditLogService.LogRBACEvent(ctx, userID, roleID, "revoke", map[string]string{"role": role.Name}, nil, "success"); err != nil {
		return apperror.InternalServerError("Failed to log audit event", err)
	}

	return nil
}

func (s *RBACService) GetUserRoles(ctx context.Context, userID uuid.UUID) (*dto.UserRolesResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalServerError("Failed to fetch user", err)
	}

	if user == nil {
		return nil, apperror.NotFound("User not found")
	}

	roles := make([]dto.RoleResponse, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = dto.RoleResponse{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
		}
	}

	permissions, err := s.permissionRepo.GetByUserID(ctx, userID)
	if err != nil {
		permissions = []*entity.Permission{}
	}

	permResponses := make([]dto.PermissionResponse, len(permissions))
	for i, perm := range permissions {
		permResponses[i] = dto.PermissionResponse{
			ID:       perm.ID,
			Name:     perm.Name,
			Resource: perm.Resource,
			Action:   perm.Action,
		}
	}

	return &dto.UserRolesResponse{
		UserID:      user.ID,
		Email:       user.Email,
		Roles:       roles,
		Permissions: permResponses,
	}, nil
}
