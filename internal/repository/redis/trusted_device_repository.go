package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hatuan/auth-service/internal/domain/entity"
	"github.com/hatuan/auth-service/internal/domain/repository"
	"github.com/redis/go-redis/v9"
)

type trustedDeviceRepository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewTrustedDeviceRepository(client *redis.Client, ttl time.Duration) repository.TrustedDeviceRepository {
	return &trustedDeviceRepository{
		client: client,
		ttl:    ttl,
	}
}

func (r *trustedDeviceRepository) Save(ctx context.Context, device *entity.TrustedDevice) error {
	data, err := json.Marshal(device)
	if err != nil {
		return err
	}

	key := r.deviceKey(device.UserID, device.Token)
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *trustedDeviceRepository) Exists(ctx context.Context, userID uuid.UUID, token string) (bool, error) {
	key := r.deviceKey(userID, token)
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *trustedDeviceRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.TrustedDevice, error) {
	pattern := r.userDevicePattern(userID)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return []*entity.TrustedDevice{}, nil
	}

	devices := make([]*entity.TrustedDevice, 0, len(keys))
	for _, key := range keys {
		val, err := r.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var device entity.TrustedDevice
		if err := json.Unmarshal([]byte(val), &device); err != nil {
			continue
		}

		devices = append(devices, &device)
	}

	return devices, nil
}

func (r *trustedDeviceRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	pattern := r.userDevicePattern(userID)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	return r.client.Del(ctx, keys...).Err()
}

func (r *trustedDeviceRepository) deviceKey(userID uuid.UUID, token string) string {
	return fmt.Sprintf("trusted_device:%s:%s", userID.String(), token)
}

func (r *trustedDeviceRepository) userDevicePattern(userID uuid.UUID) string {
	return fmt.Sprintf("trusted_device:%s:*", userID.String())
}
