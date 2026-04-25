package hash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBcrypt_HashPassword(t *testing.T) {
	password := "my-secure-password-12345"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestBcrypt_ComparePasswordValid(t *testing.T) {
	password := "my-secure-password-12345"

	hash, err := HashPassword(password)
	require.NoError(t, err)

	match, err := ComparePassword(hash, password)
	require.NoError(t, err)
	assert.True(t, match)
}

func TestBcrypt_ComparePasswordInvalid(t *testing.T) {
	password := "my-secure-password-12345"
	wrongPassword := "wrong-password-12345"

	hash, err := HashPassword(password)
	require.NoError(t, err)

	match, err := ComparePassword(hash, wrongPassword)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestBcrypt_DifferentHashesSamePassword(t *testing.T) {
	password := "my-secure-password-12345"

	hash1, err := HashPassword(password)
	require.NoError(t, err)

	hash2, err := HashPassword(password)
	require.NoError(t, err)

	// Different hashes for same password (due to random salt)
	assert.NotEqual(t, hash1, hash2)

	// But both should match
	match1, err := ComparePassword(hash1, password)
	require.NoError(t, err)
	assert.True(t, match1)

	match2, err := ComparePassword(hash2, password)
	require.NoError(t, err)
	assert.True(t, match2)
}

func TestBcrypt_EmptyPassword(t *testing.T) {
	password := ""

	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	match, err := ComparePassword(hash, password)
	require.NoError(t, err)
	assert.True(t, match)
}

func TestBcrypt_LongPassword(t *testing.T) {
	// Bcrypt has a 72-byte limit
	password := "this-is-a-very-long-password-that-exceeds-72-bytes-in-total-length-abcdefghijklmnopqrstuvwxyz"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	match, err := ComparePassword(hash, password)
	require.NoError(t, err)
	assert.True(t, match)
}
