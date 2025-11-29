package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword takes a plaintext password and returns the bcrypt hash.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	return string(bytes), err
}

// CheckPassword compares a plaintext password with a stored bcrypt hash.
// Returns true if they match, false otherwise.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err == nil
}
