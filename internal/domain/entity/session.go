package entity

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	JTI       string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func NewSession(userID uuid.UUID, jti string, token string, expiresAt time.Time) *Session {
	return &Session{
		ID:        uuid.New(),
		UserID:    userID,
		JTI:       jti,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
}
