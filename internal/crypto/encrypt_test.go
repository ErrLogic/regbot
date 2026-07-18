package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key, err := NewEncryptionKey()
	require.NoError(t, err)

	plaintext := []byte("my-super-secret-password-123!")
	ciphertext, err := Encrypt(plaintext, key)
	require.NoError(t, err)
	require.NotEqual(t, plaintext, ciphertext, "ciphertext should differ from plaintext")

	decrypted, err := Decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptUniqueNonce(t *testing.T) {
	key, err := NewEncryptionKey()
	require.NoError(t, err)

	plaintext := []byte("same-password")
	c1, err := Encrypt(plaintext, key)
	require.NoError(t, err)
	c2, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	assert.NotEqual(t, c1, c2, "encryptions with different nonces should differ")
}

func TestDecryptWrongKey(t *testing.T) {
	k1, _ := NewEncryptionKey()
	k2, _ := NewEncryptionKey()

	ciphertext, err := Encrypt([]byte("secret"), k1)
	require.NoError(t, err)

	_, err = Decrypt(ciphertext, k2)
	assert.Error(t, err, "decrypt with wrong key should fail")
}

func TestDecryptTampered(t *testing.T) {
	key, _ := NewEncryptionKey()
	ciphertext, _ := Encrypt([]byte("secret"), key)

	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[len(tampered)-1] ^= 0xFF

	_, err := Decrypt(tampered, key)
	assert.Error(t, err, "decrypt of tampered ciphertext should fail")
}

func TestEncryptInvalidKey(t *testing.T) {
	_, err := Encrypt([]byte("data"), []byte("short"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key must be 32 bytes")
}

func TestDecryptInvalidKey(t *testing.T) {
	_, err := Decrypt([]byte("data"), []byte("short"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key must be 32 bytes")
}

func TestDecryptShortCiphertext(t *testing.T) {
	key, _ := NewEncryptionKey()
	_, err := Decrypt([]byte("short"), key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestNewEncryptionKey(t *testing.T) {
	k1, err := NewEncryptionKey()
	require.NoError(t, err)
	assert.Len(t, k1, 32)

	k2, _ := NewEncryptionKey()
	assert.NotEqual(t, k1, k2, "keys should be unique")
}
