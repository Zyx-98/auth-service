package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/domain/entity"
)

type PermissionRepository interface {
	Create(ctx context.Context, permission *entity.Permission) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Permission, error)
	GetByName(ctx context.Context, name string) (*entity.Permission, error)
	Update(ctx context.Context, permission *entity.Permission) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*entity.Permission, error)
	GetByRoleID(ctx context.Context, roleID uuid.UUID) ([]*entity.Permission, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Permission, error)
}
