package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatuan/auth-service/internal/domain/entity"
	"github.com/hatuan/auth-service/pkg/hash"
	"github.com/hatuan/auth-service/pkg/jwt"
)

// Mock repositories for testing
type mockUserRepo struct {
	users map[string]*entity.User
}

func (m *mockUserRepo) Create(ctx context.Context, user *entity.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*entity.User, error) {
	return m.users[id], nil
}

func (m *mockUserRepo) Update(ctx context.Context, user *entity.User) error {
	m.users[user.ID] = user
	return nil
}

type mockSessionRepo struct {
	sessions map[string]bool
}

func (m *mockSessionRepo) Store(ctx context.Context, key string, ttl time.Duration) error {
	m.sessions[key] = true
	return nil
}

func (m *mockSessionRepo) Exists(ctx context.Context, key string) (bool, error) {
	return m.sessions[key], nil
}

func (m *mockSessionRepo) Delete(ctx context.Context, key string) error {
	delete(m.sessions, key)
	return nil
}

func setupAuthService(t *testing.T) *AuthService {
	userRepo := &mockUserRepo{users: make(map[string]*entity.User)}
	sessionRepo := &mockSessionRepo{sessions: make(map[string]bool)}

	accessMaker, err := jwt.NewMaker("access-secret-key-32-chars-long")
	require.NoError(t, err)

	refreshMaker, err := jwt.NewMaker("refresh-secret-key-32-chars-lon")
	require.NoError(t, err)

	return &AuthService{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		accessMaker:    accessMaker,
		refreshMaker:   refreshMaker,
		accessDuration: 15 * time.Minute,
		refreshDuration: 7 * 24 * time.Hour,
	}
}

func TestAuthService_Register(t *testing.T) {
	svc := setupAuthService(t)
	ctx := context.Background()

	email := "test@example.com"
	password := "secure-password-12345"

	user, err := svc.Register(ctx, email, password)
	require.NoError(t, err)

	assert.NotEmpty(t, user.ID)
	assert.Equal(t, email, user.Email)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEqual(t, password, user.PasswordHash)
}

func TestAuthService_RegisterDuplicate(t *testing.T) {
	svc := setupAuthService(t)
	ctx := context.Background()

	email := "test@example.com"
	password := "secure-password-12345"

	_, err := svc.Register(ctx, email, password)
	require.NoError(t, err)

	_, err = svc.Register(ctx, email, password)
	assert.Error(t, err)
}

func TestAuthService_Login(t *testing.T) {
	svc := setupAuthService(t)
	ctx := context.Background()

	email := "test@example.com"
	password := "secure-password-12345"

	// Register first
	user, err := svc.Register(ctx, email, password)
	require.NoError(t, err)

	// Login
	response, err := svc.Login(ctx, email, password)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Token.AccessToken)
	assert.NotEmpty(t, response.Token.RefreshToken)
	assert.False(t, response.Requires2FA)
	assert.Equal(t, user.ID, response.UserID)
}

func TestAuthService_LoginInvalidPassword(t *testing.T) {
	svc := setupAuthService(t)
	ctx := context.Background()

	email := "test@example.com"
	password := "secure-password-12345"

	// Register first
	_, err := svc.Register(ctx, email, password)
	require.NoError(t, err)

	// Try login with wrong password
	_, err = svc.Login(ctx, email, "wrong-password")
	assert.Error(t, err)
}

func TestAuthService_LoginUserNotFound(t *testing.T) {
	svc := setupAuthService(t)
	ctx := context.Background()

	_, err := svc.Login(ctx, "nonexistent@example.com", "password")
	assert.Error(t, err)
}

func TestAuthService_Introspect(t *testing.T) {
	svc := setupAuthService(t)
	ctx := context.Background()

	email := "test@example.com"
	password := "secure-password-12345"

	// Register and login
	user, err := svc.Register(ctx, email, password)
	require.NoError(t, err)

	response, err := svc.Login(ctx, email, password)
	require.NoError(t, err)

	// Introspect the token
	introspect, err := svc.Introspect(ctx, response.Token.AccessToken)
	require.NoError(t, err)

	assert.True(t, introspect.Valid)
	assert.Equal(t, user.ID, introspect.UserID)
	assert.Equal(t, email, introspect.Email)
	assert.NotZero(t, introspect.ExpiresAt)
}

func TestAuthService_IntrospectInvalidToken(t *testing.T) {
	svc := setupAuthService(t)
	ctx := context.Background()

	introspect, err := svc.Introspect(ctx, "invalid.token.here")
	require.NoError(t, err)

	assert.False(t, introspect.Valid)
}

func TestAuthService_RefreshToken(t *testing.T) {
	svc := setupAuthService(t)
	ctx := context.Background()

	email := "test@example.com"
	password := "secure-password-12345"

	// Register and login
	_, err := svc.Register(ctx, email, password)
	require.NoError(t, err)

	response, err := svc.Login(ctx, email, password)
	require.NoError(t, err)

	// Refresh the token
	newTokens, err := svc.RefreshToken(ctx, response.Token.RefreshToken)
	require.NoError(t, err)

	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEmpty(t, newTokens.RefreshToken)
	assert.NotEqual(t, response.Token.AccessToken, newTokens.AccessToken)
}

func TestAuthService_Logout(t *testing.T) {
	svc := setupAuthService(t)
	ctx := context.Background()

	email := "test@example.com"
	password := "secure-password-12345"

	// Register and login
	_, err := svc.Register(ctx, email, password)
	require.NoError(t, err)

	response, err := svc.Login(ctx, email, password)
	require.NoError(t, err)

	// Logout
	err = svc.Logout(ctx, response.Token.RefreshToken)
	require.NoError(t, err)

	// Token should be revoked
	newTokens, err := svc.RefreshToken(ctx, response.Token.RefreshToken)
	assert.Error(t, err)
	assert.Nil(t, newTokens)
}
