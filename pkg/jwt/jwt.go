package jwt

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Claims struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	JTI         string    `json:"jti"`
	TokenType   TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

// UnmarshalJSON handles custom unmarshaling of Claims to properly convert UserID string to UUID
func (c *Claims) UnmarshalJSON(data []byte) error {
	type Alias Claims
	aux := &struct {
		UserID string `json:"user_id"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.UserID != "" {
		parsedUUID, err := uuid.Parse(aux.UserID)
		if err != nil {
			return fmt.Errorf("failed to parse user_id as UUID: %w", err)
		}
		c.UserID = parsedUUID
	}

	return nil
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type Maker struct {
	accessSecret  string
	refreshSecret string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewMaker(accessSecret, refreshSecret string, accessExpiry, refreshExpiry time.Duration) *Maker {
	return &Maker{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (m *Maker) CreateTokenPair(userID uuid.UUID, email string, roles, permissions []string) (*TokenPair, error) {
	accessToken, err := m.createToken(userID, email, roles, permissions, AccessToken, m.accessExpiry, m.accessSecret)
	if err != nil {
		return nil, err
	}

	refreshToken, err := m.createToken(userID, email, roles, permissions, RefreshToken, m.refreshExpiry, m.refreshSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (m *Maker) CreateAccessToken(userID uuid.UUID, email string, roles, permissions []string) (string, error) {
	return m.createToken(userID, email, roles, permissions, AccessToken, m.accessExpiry, m.accessSecret)
}

func (m *Maker) CreateCustomToken(userID uuid.UUID, email string, permissions []string, expiry time.Duration, secret string) (string, error) {
	if secret == "" {
		secret = m.accessSecret
	}
	return m.createToken(userID, email, []string{}, permissions, AccessToken, expiry, secret)
}

func (m *Maker) CreateRefreshToken(userID uuid.UUID, email string, roles, permissions []string) (string, error) {
	return m.createToken(userID, email, roles, permissions, RefreshToken, m.refreshExpiry, m.refreshSecret)
}

func (m *Maker) createToken(userID uuid.UUID, email string, roles, permissions []string, tokenType TokenType, expiry time.Duration, secret string) (string, error) {
	jti := uuid.New().String()
	now := time.Now()

	claims := &Claims{
		UserID:      userID,
		Email:       email,
		Roles:       roles,
		Permissions: permissions,
		JTI:         jti,
		TokenType:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "auth-service",
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (m *Maker) VerifyAccessToken(tokenString string) (*Claims, error) {
	return m.verifyToken(tokenString, AccessToken, m.accessSecret)
}

func (m *Maker) VerifyRefreshToken(tokenString string) (*Claims, error) {
	return m.verifyToken(tokenString, RefreshToken, m.refreshSecret)
}

func (m *Maker) verifyToken(tokenString string, expectedType TokenType, secret string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	if claims.TokenType != expectedType {
		return nil, fmt.Errorf("invalid token type, expected %s but got %s", expectedType, claims.TokenType)
	}

	return claims, nil
}

func (m *Maker) ExtractJTI(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return nil, nil
	})

	if err != nil && token == nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok {
		return claims.JTI, nil
	}

	return "", fmt.Errorf("invalid claims type")
}
