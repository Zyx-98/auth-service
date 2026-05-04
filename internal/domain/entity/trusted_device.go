package entity

import (
	"crypto/md5"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

type TrustedDevice struct {
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewTrustedDevice(userID uuid.UUID, token, userAgent, ipAddress, name string, expiresAt time.Time) *TrustedDevice {
	return &TrustedDevice{
		Token:     token,
		UserID:    userID,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		Name:      name,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
}

func (d *TrustedDevice) Fingerprint() string {
	h := md5.New()
	_, _ = io.WriteString(h, d.UserAgent+d.IPAddress)
	return fmt.Sprintf("%x", h.Sum(nil))
}
