package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateBackupCodes generates a specified number of cryptographically secure 8-character backup codes.
func GenerateBackupCodes(count int) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be greater than 0")
	}

	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const codeLength = 8

	codes := make([]string, count)

	for i := range count {
		code, err := generateSecureCode(charset, codeLength)
		if err != nil {
			return nil, fmt.Errorf("failed to generate backup code %d: %w", i+1, err)
		}
		codes[i] = code
	}

	return codes, nil
}

// generateSecureCode generates a single cryptographically secure random code.
func generateSecureCode(charset string, length int) (string, error) {
	code := make([]byte, length)
	charsetSize := big.NewInt(int64(len(charset)))

	for i := range length {
		randomIndex, err := rand.Int(rand.Reader, charsetSize)
		if err != nil {
			return "", fmt.Errorf("failed to generate random character: %w", err)
		}
		code[i] = charset[randomIndex.Int64()]
	}

	return string(code), nil
}

// FormatBackupCode formats a backup code for display (adds spaces for readability).
func FormatBackupCode(code string) string {
	if len(code) != 8 {
		return code
	}
	return code[:4] + " " + code[4:]
}

// ValidateBackupCodeFormat checks if a code matches the expected backup code format.
func ValidateBackupCodeFormat(code string) bool {
	if len(code) != 8 && len(code) != 9 {
		return false
	}

	if len(code) == 9 && code[4] == ' ' {
		code = code[:4] + code[5:]
	}

	for _, char := range code {
		if !((char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}

	return true
}
