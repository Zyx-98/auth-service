package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/domain/entity"
)

type SessionRepository interface {
	Save(ctx context.Context, session *entity.Session) error
	GetByJTI(ctx context.Context, jti string) (*entity.Session, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Session, error)
	DeleteByJTI(ctx context.Context, jti string) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	Exists(ctx context.Context, jti string) (bool, error)
}
