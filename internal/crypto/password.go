package crypto

import (
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword returns a bcrypt hash of password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(bytes), err
}

// CheckPassword compares a bcrypt hash against password.
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
