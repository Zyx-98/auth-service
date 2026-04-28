package entity

import (
	"time"

	"github.com/google/uuid"
)

type TrustedDevice struct {
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewTrustedDevice(userID uuid.UUID, token, userAgent, ipAddress string, expiresAt time.Time) *TrustedDevice {
	return &TrustedDevice{
		Token:     token,
		UserID:    userID,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
}
