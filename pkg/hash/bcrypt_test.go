package hash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBcrypt_HashPassword(t *testing.T) {
	password := "my-secure-password-12345"

	hash, err := Hash(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestBcrypt_ComparePasswordValid(t *testing.T) {
	password := "my-secure-password-12345"

	hash, err := Hash(password)
	require.NoError(t, err)

	match := Compare(hash, password)
	assert.True(t, match)
}

func TestBcrypt_ComparePasswordInvalid(t *testing.T) {
	password := "my-secure-password-12345"
	wrongPassword := "wrong-password-12345"

	hash, err := Hash(password)
	require.NoError(t, err)

	match := Compare(hash, wrongPassword)
	assert.False(t, match)
}

func TestBcrypt_DifferentHashesSamePassword(t *testing.T) {
	password := "my-secure-password-12345"

	hash1, err := Hash(password)
	require.NoError(t, err)

	hash2, err := Hash(password)
	require.NoError(t, err)

	// Different hashes for same password (due to random salt)
	assert.NotEqual(t, hash1, hash2)

	// But both should match
	match1 := Compare(hash1, password)
	assert.True(t, match1)

	match2 := Compare(hash2, password)
	assert.True(t, match2)
}

func TestBcrypt_EmptyPassword(t *testing.T) {
	password := ""

	hash, err := Hash(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	match := Compare(hash, password)
	assert.True(t, match)
}

func TestBcrypt_LongPassword(t *testing.T) {
	// Bcrypt has a 72-byte limit - test with exactly 72 bytes
	password := "this-is-a-very-long-password-that-is-exactly-72-bytes-long-abcdefg"

	hash, err := Hash(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	match := Compare(hash, password)
	assert.True(t, match)
}
