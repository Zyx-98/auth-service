package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTMaker_IssueAccessToken(t *testing.T) {
	maker, err := NewMaker("secret-key-32-chars-long-12345")
	require.NoError(t, err)

	userID := "550e8400-e29b-41d4-a716-446655440000"
	expiresAt := time.Now().Add(15 * time.Minute)

	token, err := maker.IssueAccessToken(userID, expiresAt)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := maker.VerifyAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.Subject)
}

func TestJWTMaker_IssueRefreshToken(t *testing.T) {
	maker, err := NewMaker("secret-key-32-chars-long-12345")
	require.NoError(t, err)

	jti := "jti-550e8400-e29b-41d4-a716"
	userID := "550e8400-e29b-41d4-a716-446655440000"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	token, err := maker.IssueRefreshToken(userID, jti, expiresAt)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := maker.VerifyRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.Subject)
	assert.Equal(t, jti, claims.ID)
}

func TestJWTMaker_ExpiredToken(t *testing.T) {
	maker, err := NewMaker("secret-key-32-chars-long-12345")
	require.NoError(t, err)

	userID := "550e8400-e29b-41d4-a716-446655440000"
	expiresAt := time.Now().Add(-1 * time.Minute)

	token, err := maker.IssueAccessToken(userID, expiresAt)
	require.NoError(t, err)

	_, err = maker.VerifyAccessToken(token)
	assert.Error(t, err)
	assert.Equal(t, ErrExpiredToken, err)
}

func TestJWTMaker_InvalidToken(t *testing.T) {
	maker, err := NewMaker("secret-key-32-chars-long-12345")
	require.NoError(t, err)

	_, err = maker.VerifyAccessToken("invalid.token.here")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestJWTMaker_DifferentSecrets(t *testing.T) {
	maker1, err := NewMaker("secret-key-1-32-chars-long-1234")
	require.NoError(t, err)

	maker2, err := NewMaker("secret-key-2-32-chars-long-1234")
	require.NoError(t, err)

	userID := "550e8400-e29b-41d4-a716-446655440000"
	expiresAt := time.Now().Add(15 * time.Minute)

	token, err := maker1.IssueAccessToken(userID, expiresAt)
	require.NoError(t, err)

	_, err = maker2.VerifyAccessToken(token)
	assert.Error(t, err)
}

func TestJWTMaker_IssueTokens(t *testing.T) {
	accessMaker, err := NewMaker("access-secret-key-32-chars-long")
	require.NoError(t, err)

	refreshMaker, err := NewMaker("refresh-secret-key-32-chars-lon")
	require.NoError(t, err)

	userID := "550e8400-e29b-41d4-a716-446655440000"
	accessDuration := 15 * time.Minute
	refreshDuration := 7 * 24 * time.Hour

	tokens, err := IssueTokens(accessMaker, refreshMaker, userID, accessDuration, refreshDuration)
	require.NoError(t, err)

	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, "Bearer", tokens.TokenType)
	assert.Equal(t, int(accessDuration.Seconds()), tokens.ExpiresIn)

	accessClaims, err := accessMaker.VerifyAccessToken(tokens.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, accessClaims.Subject)

	refreshClaims, err := refreshMaker.VerifyRefreshToken(tokens.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, userID, refreshClaims.Subject)
}
