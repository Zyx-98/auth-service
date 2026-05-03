package handler

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatuan/auth-service/internal/domain/entity"
	"github.com/hatuan/auth-service/internal/domain/repository"
	"github.com/hatuan/auth-service/internal/service"
	"github.com/hatuan/auth-service/pkg/jwt"
	"github.com/hatuan/auth-service/pkg/totp"
)

// Mock repositories for handler tests
type mockUserRepo struct {
	users map[uuid.UUID]*entity.User
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

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	return m.users[id], nil
}

func (m *mockUserRepo) GetByGoogleID(ctx context.Context, googleID string) (*entity.User, error) {
	for _, u := range m.users {
		if u.GoogleID != nil && *u.GoogleID == googleID {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) Update(ctx context.Context, user *entity.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserRepo) List(ctx context.Context, limit int, offset int) ([]*entity.User, error) {
	users := make([]*entity.User, 0)
	for _, u := range m.users {
		users = append(users, u)
	}
	return users, nil
}

type mockSessionRepo struct {
	sessions map[string]*entity.Session
}

func (m *mockSessionRepo) Save(ctx context.Context, session *entity.Session) error {
	m.sessions[session.JTI] = session
	return nil
}

func (m *mockSessionRepo) GetByJTI(ctx context.Context, jti string) (*entity.Session, error) {
	return m.sessions[jti], nil
}

func (m *mockSessionRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Session, error) {
	sessions := make([]*entity.Session, 0)
	for _, s := range m.sessions {
		if s.UserID == userID {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

func (m *mockSessionRepo) DeleteByJTI(ctx context.Context, jti string) error {
	delete(m.sessions, jti)
	return nil
}

func (m *mockSessionRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	for jti, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, jti)
		}
	}
	return nil
}

func (m *mockSessionRepo) Exists(ctx context.Context, jti string) (bool, error) {
	_, exists := m.sessions[jti]
	return exists, nil
}

type mockRoleRepo struct {
	roles map[uuid.UUID]*entity.Role
}

func (m *mockRoleRepo) Create(ctx context.Context, role *entity.Role) error {
	m.roles[role.ID] = role
	return nil
}

func (m *mockRoleRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Role, error) {
	return m.roles[id], nil
}

func (m *mockRoleRepo) GetByName(ctx context.Context, name string) (*entity.Role, error) {
	for _, r := range m.roles {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, nil
}

func (m *mockRoleRepo) Update(ctx context.Context, role *entity.Role) error {
	m.roles[role.ID] = role
	return nil
}

func (m *mockRoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.roles, id)
	return nil
}

func (m *mockRoleRepo) List(ctx context.Context) ([]*entity.Role, error) {
	roles := make([]*entity.Role, 0)
	for _, r := range m.roles {
		roles = append(roles, r)
	}
	return roles, nil
}

func (m *mockRoleRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Role, error) {
	return make([]*entity.Role, 0), nil
}

func (m *mockRoleRepo) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return nil
}

func (m *mockRoleRepo) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return nil
}

type mockPermissionRepo struct {
	permissions map[uuid.UUID]*entity.Permission
}

func (m *mockPermissionRepo) Create(ctx context.Context, permission *entity.Permission) error {
	m.permissions[permission.ID] = permission
	return nil
}

func (m *mockPermissionRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Permission, error) {
	return m.permissions[id], nil
}

func (m *mockPermissionRepo) GetByName(ctx context.Context, name string) (*entity.Permission, error) {
	for _, p := range m.permissions {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockPermissionRepo) Update(ctx context.Context, permission *entity.Permission) error {
	m.permissions[permission.ID] = permission
	return nil
}

func (m *mockPermissionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.permissions, id)
	return nil
}

func (m *mockPermissionRepo) List(ctx context.Context) ([]*entity.Permission, error) {
	permissions := make([]*entity.Permission, 0)
	for _, p := range m.permissions {
		permissions = append(permissions, p)
	}
	return permissions, nil
}

func (m *mockPermissionRepo) GetByRoleID(ctx context.Context, roleID uuid.UUID) ([]*entity.Permission, error) {
	return make([]*entity.Permission, 0), nil
}

func (m *mockPermissionRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Permission, error) {
	return make([]*entity.Permission, 0), nil
}

type mockTrustedDeviceRepo struct {
	devices map[string]*entity.TrustedDevice
}

func (m *mockTrustedDeviceRepo) Save(ctx context.Context, device *entity.TrustedDevice) error {
	key := device.UserID.String() + ":" + device.Token
	m.devices[key] = device
	return nil
}

func (m *mockTrustedDeviceRepo) Exists(ctx context.Context, userID uuid.UUID, token string) (bool, error) {
	key := userID.String() + ":" + token
	_, exists := m.devices[key]
	return exists, nil
}

func (m *mockTrustedDeviceRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.TrustedDevice, error) {
	prefix := userID.String() + ":"
	devices := make([]*entity.TrustedDevice, 0)
	for k, d := range m.devices {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			devices = append(devices, d)
		}
	}
	return devices, nil
}

func (m *mockTrustedDeviceRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	prefix := userID.String() + ":"
	for k := range m.devices {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			delete(m.devices, k)
		}
	}
	return nil
}

func (m *mockTrustedDeviceRepo) IsTrustedByUserAgentAndIP(ctx context.Context, userID uuid.UUID, userAgent, ipAddress string) (bool, error) {
	for _, device := range m.devices {
		if device.UserID == userID && device.UserAgent == userAgent && device.IPAddress == ipAddress && device.ExpiresAt.After(time.Now()) {
			return true, nil
		}
	}
	return false, nil
}

type mockAuditLogRepo struct{}

func (m *mockAuditLogRepo) Create(ctx context.Context, log *entity.AuditLog) error {
	return nil
}

func (m *mockAuditLogRepo) ListByActorID(ctx context.Context, actorID uuid.UUID, limit, offset int) ([]*entity.AuditLog, error) {
	return make([]*entity.AuditLog, 0), nil
}

func (m *mockAuditLogRepo) List(ctx context.Context, filters repository.AuditLogFilters, limit, offset int) ([]*entity.AuditLog, error) {
	return make([]*entity.AuditLog, 0), nil
}

func setupAuthHandler(t *testing.T) (*AuthHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	userRepo := &mockUserRepo{users: make(map[uuid.UUID]*entity.User)}
	roleRepo := &mockRoleRepo{roles: make(map[uuid.UUID]*entity.Role)}
	permissionRepo := &mockPermissionRepo{permissions: make(map[uuid.UUID]*entity.Permission)}
	sessionRepo := &mockSessionRepo{sessions: make(map[string]*entity.Session)}
	trustedDeviceRepo := &mockTrustedDeviceRepo{devices: make(map[string]*entity.TrustedDevice)}

	jwtMaker := jwt.NewMaker(
		"access-secret-key-32-chars-long",
		"refresh-secret-key-32-chars-lon",
		15*time.Minute,
		7*24*time.Hour,
	)

	totpManager, err := totp.NewTOTPManager("AuthService", "e90cfcd097d9116bc1a66a7ad81851db25b8556769c2ae3fa46e05fef7875edf")
	require.NoError(t, err)

	auditLogRepo := &mockAuditLogRepo{}
	auditLogSvc := service.NewAuditLogService(auditLogRepo)

	authSvc := service.NewAuthService(userRepo, roleRepo, permissionRepo, sessionRepo, trustedDeviceRepo, jwtMaker, auditLogSvc)
	totpSvc := service.NewTOTPService(totpManager, userRepo, auditLogSvc)

	handler := NewAuthHandler(authSvc, totpSvc)

	return handler, router
}

func TestAuthHandler_Setup(t *testing.T) {
	// Just verify that we can set up the handler without panicking
	handler, _ := setupAuthHandler(t)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.authService)
	assert.NotNil(t, handler.totpService)
}
