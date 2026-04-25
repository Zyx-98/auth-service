package totp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTOTP_GenerateSecret(t *testing.T) {
	secret, qrURI, err := GenerateSecret("test@example.com", "AuthService")
	require.NoError(t, err)

	assert.NotEmpty(t, secret)
	assert.NotEmpty(t, qrURI)
	assert.Contains(t, qrURI, "otpauth://totp/")
	assert.Contains(t, qrURI, "test@example.com")
}

func TestTOTP_VerifyCode(t *testing.T) {
	secret, _, err := GenerateSecret("test@example.com", "AuthService")
	require.NoError(t, err)

	// Get current TOTP code
	code, err := getCurrentCode(secret)
	require.NoError(t, err)

	// Verify the code
	valid, err := VerifyCode(secret, code)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestTOTP_InvalidCode(t *testing.T) {
	secret, _, err := GenerateSecret("test@example.com", "AuthService")
	require.NoError(t, err)

	valid, err := VerifyCode(secret, "000000")
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestTOTP_EncryptDecrypt(t *testing.T) {
	secret, _, err := GenerateSecret("test@example.com", "AuthService")
	require.NoError(t, err)

	encryptionKey := "00000000000000000000000000000000"

	encrypted, err := EncryptSecret(secret, encryptionKey)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := DecryptSecret(encrypted, encryptionKey)
	require.NoError(t, err)
	assert.Equal(t, secret, decrypted)
}

func TestTOTP_EncryptDecrypt_DifferentKey(t *testing.T) {
	secret, _, err := GenerateSecret("test@example.com", "AuthService")
	require.NoError(t, err)

	key1 := "00000000000000000000000000000000"
	key2 := "11111111111111111111111111111111"

	encrypted, err := EncryptSecret(secret, key1)
	require.NoError(t, err)

	_, err = DecryptSecret(encrypted, key2)
	assert.Error(t, err)
}

func TestTOTP_TimeSensitive(t *testing.T) {
	secret, _, err := GenerateSecret("test@example.com", "AuthService")
	require.NoError(t, err)

	code1, err := getCurrentCode(secret)
	require.NoError(t, err)

	valid, err := VerifyCode(secret, code1)
	require.NoError(t, err)
	assert.True(t, valid)

	// Code should be valid within 30-second window
	valid, err = VerifyCode(secret, code1)
	require.NoError(t, err)
	assert.True(t, valid)
}

// Helper function to get current TOTP code for testing
func getCurrentCode(secret string) (string, error) {
	totp, err := totp.NewKeyFromURL("otpauth://totp/test?secret=" + secret)
	if err != nil {
		return "", err
	}
	return totp.OTP()
}
