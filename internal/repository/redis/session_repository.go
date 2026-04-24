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

type sessionRepository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewSessionRepository(client *redis.Client, ttl time.Duration) repository.SessionRepository {
	return &sessionRepository{
		client: client,
		ttl:    ttl,
	}
}

func (r *sessionRepository) Save(ctx context.Context, session *entity.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	key := r.sessionKey(session.JTI)
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *sessionRepository) GetByJTI(ctx context.Context, jti string) (*entity.Session, error) {
	key := r.sessionKey(jti)
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var session entity.Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *sessionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Session, error) {
	pattern := r.userSessionPattern(userID)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return []*entity.Session{}, nil
	}

	var sessions []*entity.Session
	for _, key := range keys {
		data, err := r.client.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return nil, err
		}

		var session entity.Session
		if err := json.Unmarshal([]byte(data), &session); err != nil {
			continue
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

func (r *sessionRepository) DeleteByJTI(ctx context.Context, jti string) error {
	key := r.sessionKey(jti)
	return r.client.Del(ctx, key).Err()
}

func (r *sessionRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	pattern := r.userSessionPattern(userID)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	return r.client.Del(ctx, keys...).Err()
}

func (r *sessionRepository) Exists(ctx context.Context, jti string) (bool, error) {
	key := r.sessionKey(jti)
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *sessionRepository) sessionKey(jti string) string {
	return fmt.Sprintf("refresh:%s", jti)
}

func (r *sessionRepository) userSessionPattern(userID uuid.UUID) string {
	return fmt.Sprintf("refresh:%s:*", userID.String())
}
