package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatuan/auth-service/internal/domain/entity"
	"github.com/hatuan/auth-service/internal/service"
	"github.com/hatuan/auth-service/pkg/jwt"
)

// Mock repositories for handler tests
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

func setupAuthHandler(t *testing.T) (*AuthHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	userRepo := &mockUserRepo{users: make(map[string]*entity.User)}
	sessionRepo := &mockSessionRepo{sessions: make(map[string]bool)}

	accessMaker, err := jwt.NewMaker("access-secret-key-32-chars-long")
	require.NoError(t, err)

	refreshMaker, err := jwt.NewMaker("refresh-secret-key-32-chars-lon")
	require.NoError(t, err)

	authSvc := &service.AuthService{
		UserRepo:        userRepo,
		SessionRepo:     sessionRepo,
		AccessMaker:     accessMaker,
		RefreshMaker:    refreshMaker,
		AccessDuration:  15 * time.Minute,
		RefreshDuration: 7 * 24 * time.Hour,
	}

	handler := &AuthHandler{authService: authSvc}

	return handler, router
}

func TestAuthHandler_Register(t *testing.T) {
	handler, router := setupAuthHandler(t)
	router.POST("/auth/register", handler.Register)

	payload := map[string]string{
		"email":             "test@example.com",
		"password":          "secure-password-12345",
		"password_confirm": "secure-password-12345",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "user registered successfully", response["message"])
}

func TestAuthHandler_RegisterPasswordMismatch(t *testing.T) {
	handler, router := setupAuthHandler(t)
	router.POST("/auth/register", handler.Register)

	payload := map[string]string{
		"email":             "test@example.com",
		"password":          "secure-password-12345",
		"password_confirm": "different-password",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Login(t *testing.T) {
	handler, router := setupAuthHandler(t)
	router.POST("/auth/register", handler.Register)
	router.POST("/auth/login", handler.Login)

	// Register first
	regPayload := map[string]string{
		"email":             "test@example.com",
		"password":          "secure-password-12345",
		"password_confirm": "secure-password-12345",
	}

	body, _ := json.Marshal(regPayload)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Login
	loginPayload := map[string]string{
		"email":    "test@example.com",
		"password": "secure-password-12345",
	}

	body, _ = json.Marshal(loginPayload)
	req = httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.NotEmpty(t, data["token"])
	assert.False(t, data["requires_2fa"].(bool))
}

func TestAuthHandler_LoginInvalidCredentials(t *testing.T) {
	handler, router := setupAuthHandler(t)
	router.POST("/auth/register", handler.Register)
	router.POST("/auth/login", handler.Login)

	// Register
	regPayload := map[string]string{
		"email":             "test@example.com",
		"password":          "secure-password-12345",
		"password_confirm": "secure-password-12345",
	}

	body, _ := json.Marshal(regPayload)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Try login with wrong password
	loginPayload := map[string]string{
		"email":    "test@example.com",
		"password": "wrong-password",
	}

	body, _ = json.Marshal(loginPayload)
	req = httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Introspect(t *testing.T) {
	handler, router := setupAuthHandler(t)
	router.POST("/auth/register", handler.Register)
	router.POST("/auth/login", handler.Login)
	router.POST("/auth/introspect", handler.Introspect)

	// Register and login
	regPayload := map[string]string{
		"email":             "test@example.com",
		"password":          "secure-password-12345",
		"password_confirm": "secure-password-12345",
	}

	body, _ := json.Marshal(regPayload)
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Get token
	loginPayload := map[string]string{
		"email":    "test@example.com",
		"password": "secure-password-12345",
	}

	body, _ = json.Marshal(loginPayload)
	req = httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	data := loginResponse["data"].(map[string]interface{})
	tokenData := data["token"].(map[string]interface{})
	token := tokenData["access_token"].(string)

	// Introspect
	introspectPayload := map[string]string{
		"token": token,
	}

	body, _ = json.Marshal(introspectPayload)
	req = httptest.NewRequest("POST", "/auth/introspect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var introspectResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &introspectResponse)

	introspectData := introspectResponse["data"].(map[string]interface{})
	assert.True(t, introspectData["valid"].(bool))
	assert.Equal(t, "test@example.com", introspectData["email"].(string))
	assert.NotEmpty(t, introspectData["user_id"])
}
