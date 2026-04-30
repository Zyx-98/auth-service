package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hatuan/auth-service/pkg/jwt"
	"github.com/stretchr/testify/assert"
)

func setupTestJWTMaker() *jwt.Maker {
	return jwt.NewMaker("test-access-secret", "test-refresh-secret", 15*time.Minute, 7*24*time.Hour)
}

func TestAuthMiddleware_RejectsTemporaryTokens(t *testing.T) {
	jwtMaker := setupTestJWTMaker()
	userID := uuid.New()
	email := "test@example.com"

	// Create a temporary token with only 2fa:verify permission
	tempToken, err := jwtMaker.CreateCustomToken(userID, email, []string{"2fa:verify"}, 5*time.Minute, "")
	assert.NoError(t, err)

	router := gin.New()
	router.Use(AuthMiddleware(jwtMaker))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tempToken))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Temporary token cannot access this endpoint")
}

func TestAuthMiddleware_AcceptsFullAccessTokens(t *testing.T) {
	jwtMaker := setupTestJWTMaker()
	userID := uuid.New()
	email := "test@example.com"
	roles := []string{"user"}
	permissions := []string{"user:read", "user:write"}

	// Create a full access token
	accessToken, err := jwtMaker.CreateAccessToken(userID, email, roles, permissions)
	assert.NoError(t, err)

	router := gin.New()
	router.Use(AuthMiddleware(jwtMaker))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	jwtMaker := setupTestJWTMaker()

	router := gin.New()
	router.Use(AuthMiddleware(jwtMaker))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Missing authorization header")
}

func TestAuthMiddleware_InvalidTokenFormat(t *testing.T) {
	jwtMaker := setupTestJWTMaker()

	router := gin.New()
	router.Use(AuthMiddleware(jwtMaker))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid authorization header format")
}

func TestTwoFAMiddleware_AcceptsTemporaryTokens(t *testing.T) {
	jwtMaker := setupTestJWTMaker()
	userID := uuid.New()
	email := "test@example.com"

	// Create a temporary token with only 2fa:verify permission
	tempToken, err := jwtMaker.CreateCustomToken(userID, email, []string{"2fa:verify"}, 5*time.Minute, "")
	assert.NoError(t, err)

	router := gin.New()
	router.Use(TwoFAMiddleware(jwtMaker))
	router.POST("/2fa/verify-login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "verified"})
	})

	req := httptest.NewRequest("POST", "/2fa/verify-login", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tempToken))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "verified")
}

func TestTwoFAMiddleware_RejectsFullAccessTokens(t *testing.T) {
	jwtMaker := setupTestJWTMaker()
	userID := uuid.New()
	email := "test@example.com"
	roles := []string{"user"}
	permissions := []string{"user:read"}

	// Create a full access token without 2fa:verify permission
	accessToken, err := jwtMaker.CreateAccessToken(userID, email, roles, permissions)
	assert.NoError(t, err)

	router := gin.New()
	router.Use(TwoFAMiddleware(jwtMaker))
	router.POST("/2fa/verify-login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "verified"})
	})

	req := httptest.NewRequest("POST", "/2fa/verify-login", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Insufficient permissions for 2FA verification")
}

func TestTwoFAMiddleware_RejectsTokensWithoutPermission(t *testing.T) {
	jwtMaker := setupTestJWTMaker()
	userID := uuid.New()
	email := "test@example.com"

	// Create a token with no permissions
	invalidToken, err := jwtMaker.CreateCustomToken(userID, email, []string{}, 5*time.Minute, "")
	assert.NoError(t, err)

	router := gin.New()
	router.Use(TwoFAMiddleware(jwtMaker))
	router.POST("/2fa/verify-login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "verified"})
	})

	req := httptest.NewRequest("POST", "/2fa/verify-login", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", invalidToken))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Insufficient permissions for 2FA verification")
}

func TestIsTempToken(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		expected    bool
	}{
		{
			name:        "Valid temp token",
			permissions: []string{"2fa:verify"},
			expected:    true,
		},
		{
			name:        "Empty permissions",
			permissions: []string{},
			expected:    false,
		},
		{
			name:        "Multiple permissions",
			permissions: []string{"2fa:verify", "user:read"},
			expected:    false,
		},
		{
			name:        "Different permission",
			permissions: []string{"user:read"},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTempToken(tt.permissions)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		required    string
		expected    bool
	}{
		{
			name:        "Permission exists",
			permissions: []string{"2fa:verify", "user:read"},
			required:    "2fa:verify",
			expected:    true,
		},
		{
			name:        "Permission not found",
			permissions: []string{"user:read"},
			required:    "2fa:verify",
			expected:    false,
		},
		{
			name:        "Empty permissions",
			permissions: []string{},
			required:    "2fa:verify",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasPermission(tt.permissions, tt.required)
			assert.Equal(t, tt.expected, result)
		})
	}
}
