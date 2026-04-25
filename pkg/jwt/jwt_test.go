package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTMaker_CreateTokenPair(t *testing.T) {
	maker := NewMaker(
		"access-secret-key-32-chars-long",
		"refresh-secret-key-32-chars-lon",
		15*time.Minute,
		7*24*time.Hour,
	)

	userID := "550e8400-e29b-41d4-a716-446655440000"
	email := "test@example.com"
	roles := []string{"user"}
	permissions := []string{"read"}

	pair, err := maker.CreateTokenPair(parseUUID(t, userID), email, roles, permissions)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
}

func TestJWTMaker_CreateAccessToken(t *testing.T) {
	maker := NewMaker(
		"access-secret-key-32-chars-long",
		"refresh-secret-key-32-chars-lon",
		15*time.Minute,
		7*24*time.Hour,
	)

	userID := "550e8400-e29b-41d4-a716-446655440000"
	email := "test@example.com"
	roles := []string{"user"}
	permissions := []string{"read"}

	token, err := maker.CreateAccessToken(parseUUID(t, userID), email, roles, permissions)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := maker.VerifyAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.Subject)
}

func TestJWTMaker_CreateRefreshToken(t *testing.T) {
	maker := NewMaker(
		"access-secret-key-32-chars-long",
		"refresh-secret-key-32-chars-lon",
		15*time.Minute,
		7*24*time.Hour,
	)

	userID := "550e8400-e29b-41d4-a716-446655440000"
	email := "test@example.com"
	roles := []string{"user"}
	permissions := []string{"read"}

	token, err := maker.CreateRefreshToken(parseUUID(t, userID), email, roles, permissions)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := maker.VerifyRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.Subject)
}

func TestJWTMaker_VerifyAccessToken_Expired(t *testing.T) {
	maker := NewMaker(
		"access-secret-key-32-chars-long",
		"refresh-secret-key-32-chars-lon",
		-1*time.Minute,
		7*24*time.Hour,
	)

	userID := "550e8400-e29b-41d4-a716-446655440000"
	email := "test@example.com"
	roles := []string{"user"}
	permissions := []string{"read"}

	token, err := maker.CreateAccessToken(parseUUID(t, userID), email, roles, permissions)
	require.NoError(t, err)

	_, err = maker.VerifyAccessToken(token)
	assert.Error(t, err)
}

func TestJWTMaker_VerifyAccessToken_Invalid(t *testing.T) {
	maker := NewMaker(
		"access-secret-key-32-chars-long",
		"refresh-secret-key-32-chars-lon",
		15*time.Minute,
		7*24*time.Hour,
	)

	_, err := maker.VerifyAccessToken("invalid.token.here")
	assert.Error(t, err)
}

func TestJWTMaker_DifferentSecrets(t *testing.T) {
	maker1 := NewMaker(
		"access-secret-1-32-chars-long",
		"refresh-secret-1-32-chars-long",
		15*time.Minute,
		7*24*time.Hour,
	)

	maker2 := NewMaker(
		"access-secret-2-32-chars-long",
		"refresh-secret-2-32-chars-long",
		15*time.Minute,
		7*24*time.Hour,
	)

	userID := "550e8400-e29b-41d4-a716-446655440000"
	email := "test@example.com"
	roles := []string{"user"}
	permissions := []string{"read"}

	token, err := maker1.CreateAccessToken(parseUUID(t, userID), email, roles, permissions)
	require.NoError(t, err)

	_, err = maker2.VerifyAccessToken(token)
	assert.Error(t, err)
}

func parseUUID(t *testing.T, uuidStr string) uuid.UUID {
	id, err := uuid.Parse(uuidStr)
	require.NoError(t, err)
	return id
}
