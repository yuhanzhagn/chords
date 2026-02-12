package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashPasswordAndCheck(t *testing.T) {
	password := "mySecret123!"

	// Hash the password
	hash, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// Check the correct password
	ok := CheckPasswordHash(password, hash)
	require.True(t, ok)

	// Check a wrong password
	ok = CheckPasswordHash("wrongPassword", hash)
	require.False(t, ok)
}

func TestHashPassword_UniqueHash(t *testing.T) {
	password := "repeatMe"

	// Two hashes of the same password should NOT be equal due to salt
	hash1, err := HashPassword(password)
	require.NoError(t, err)

	hash2, err := HashPassword(password)
	require.NoError(t, err)

	require.NotEqual(t, hash1, hash2)
}
