package dto

import "github.com/google/uuid"

type RegisterRequest struct {
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8"`
	PasswordConfirm string `json:"password_confirm" validate:"required,eqfield=Password"`
}

type LoginRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required"`
	DeviceToken string `json:"device_token,omitempty"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type LoginResponse struct {
	RequiresTwoFA bool           `json:"requires_2fa,omitempty"`
	TempToken     string         `json:"temp_token,omitempty"`
	Token         *TokenResponse `json:"token,omitempty"`
	DeviceToken   string         `json:"device_token,omitempty"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type IntrospectRequest struct {
	Token string `json:"token" validate:"required"`
}

type IntrospectResponse struct {
	Valid       bool     `json:"valid"`
	UserID      uuid.UUID `json:"user_id,omitempty"`
	Email       string   `json:"email,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	ExpiresAt   int64    `json:"expires_at,omitempty"`
}

type UserProfileResponse struct {
	ID      uuid.UUID `json:"id"`
	Email   string    `json:"email"`
	Roles   []string  `json:"roles"`
	TOTPEnabled bool   `json:"totp_enabled"`
}
