package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	password := "testPassword123"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
	assert.Contains(t, hash, "$")
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	password := ""

	hash, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	assert.Contains(t, hash, "$")
}

func TestHashPassword_DifferentPasswords(t *testing.T) {
	password1 := "password1"
	password2 := "password2"

	hash1, err1 := HashPassword(password1)
	hash2, err2 := HashPassword(password2)

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NotEmpty(t, hash1)
	require.NotEmpty(t, hash2)
	assert.NotEqual(t, hash1, hash2)
}

func TestHashPassword_SamePasswordDifferentHashes(t *testing.T) {
	password := "samePassword"

	hash1, err1 := HashPassword(password)
	hash2, err2 := HashPassword(password)

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NotEmpty(t, hash1)
	require.NotEmpty(t, hash2)

	assert.NotEqual(t, hash1, hash2)
}

func TestHashPassword_LongPassword(t *testing.T) {
	password := "thisIsAVeryLongPasswordThatExceedsNormalLengthsAndContainsNumbers"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	assert.Contains(t, hash, "$")
}

func TestCheckPassword_CorrectPassword(t *testing.T) {
	password := "correctPassword123"

	hash, err := HashPassword(password)
	require.NoError(t, err)

	isValid := CheckPassword(password, hash)
	assert.True(t, isValid)
}

func TestCheckPassword_IncorrectPassword(t *testing.T) {
	password := "correctPassword123"
	wrongPassword := "wrongPassword456"

	hash, err := HashPassword(password)
	require.NoError(t, err)

	isValid := CheckPassword(wrongPassword, hash)
	assert.False(t, isValid)
}

func TestCheckPassword_EmptyPassword(t *testing.T) {
	password := ""
	hash, err := HashPassword(password)
	require.NoError(t, err)

	isValid := CheckPassword(password, hash)
	assert.True(t, isValid)

	isValid = CheckPassword("nonEmpty", hash)
	assert.False(t, isValid)
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	password := "testPassword"
	invalidHash := "invalid-hash-format"

	isValid := CheckPassword(password, invalidHash)
	assert.False(t, isValid)
}

func TestCheckPassword_EmptyHash(t *testing.T) {
	password := "testPassword"

	isValid := CheckPassword(password, "")
	assert.False(t, isValid)
}

func TestPasswordHashing_RoundTrip(t *testing.T) {
	passwords := []string{
		"simple",
		"complex123!@#",
		"veryLongPasswordWithManyCharactersAndNumbers123456789",
		"password with spaces",
		"密码",
		"пароль",
		"🔐🔑",
	}

	for _, password := range passwords {
		t.Run("password_"+password, func(t *testing.T) {
			hash, err := HashPassword(password)
			require.NoError(t, err)
			require.NotEmpty(t, hash)

			isValid := CheckPassword(password, hash)
			assert.True(t, isValid, "Password should match its hash")

			isInvalid := CheckPassword(password+"wrong", hash)
			assert.False(t, isInvalid, "Wrong password should not match")
		})
	}
}
