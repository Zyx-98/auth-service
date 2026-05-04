package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/Zyx-98/auth-service/internal/domain/entity"
)

type TrustedDeviceRepository interface {
	Save(ctx context.Context, device *entity.TrustedDevice) error
	Exists(ctx context.Context, userID uuid.UUID, token string) (bool, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.TrustedDevice, error)
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	IsTrustedByUserAgentAndIP(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string) (bool, error)
}
