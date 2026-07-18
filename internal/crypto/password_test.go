package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("my-password")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "my-password", hash, "hash should not equal plaintext")
}

func TestCheckPasswordSuccess(t *testing.T) {
	hash, _ := HashPassword("correct-password")
	err := CheckPassword(hash, "correct-password")
	assert.NoError(t, err)
}

func TestCheckPasswordWrong(t *testing.T) {
	hash, _ := HashPassword("correct-password")
	err := CheckPassword(hash, "wrong-password")
	assert.Error(t, err)
}

func TestHashPasswordEmpty(t *testing.T) {
	hash, err := HashPassword("")
	require.NoError(t, err)
	err = CheckPassword(hash, "")
	assert.NoError(t, err)
}

func TestHashPasswordUniqueness(t *testing.T) {
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")
	assert.NotEqual(t, h1, h2, "bcrypt salts should produce different hashes")
}
