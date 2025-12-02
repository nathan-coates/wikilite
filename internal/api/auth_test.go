package api

import (
	"context"
	"errors"
	"testing"
	"time"
	"wikilite/pkg/models"
	"wikilite/pkg/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleLoginToken_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	input := &LoginInput{}
	input.Body.Email = user.Email
	input.Body.Password = password

	resp, err := server.handleLoginToken(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Body.Token)

	token, err := jwt.Parse(resp.Body.Token, func(token *jwt.Token) (any, error) {
		return server.jwtSecret, nil
	})
	require.NoError(t, err)
	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, user.Email, claims["email"])
	assert.Equal(t, float64(user.Role), claims["role"])
}

func TestHandleLoginToken_InvalidCredentials(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	input := &LoginInput{}
	input.Body.Email = "test@example.com"
	input.Body.Password = "wrong"

	_, err := server.handleLoginToken(context.Background(), input)
	require.Error(t, err)
	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 401, humaErr.Status)
}

func TestHandleLoginToken_DisabledUser(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{
		Name:     "Disabled User",
		Email:    "disabled@user.com",
		Role:     models.READ,
		Disabled: true,
	}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	input := &LoginInput{}
	input.Body.Email = user.Email
	input.Body.Password = password

	_, err = server.handleLoginToken(context.Background(), input)
	require.Error(t, err)
	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}

func TestHandleLoginToken_ExternalUser(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{
		Name:       "External User",
		Email:      "external@user.com",
		Role:       models.READ,
		IsExternal: true,
	}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	input := &LoginInput{}
	input.Body.Email = user.Email
	input.Body.Password = "any"

	_, err = server.handleLoginToken(context.Background(), input)
	require.Error(t, err)
	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 400, humaErr.Status)
}

func TestHandleStartOTPEnrollment_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), userContextKey, user)

	input := &OTPStartEnrollmentInput{}
	input.Body.Password = password

	resp, err := server.handleStartOTPEnrollment(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Body.Code)
	assert.NotEmpty(t, resp.Body.QRCode)
	assert.Equal(t, "Test Wiki", resp.Body.Issuer)
	assert.Len(t, resp.Body.BackupCodes, 10)

	for _, code := range resp.Body.BackupCodes {
		assert.Len(t, code, 9)
		assert.Contains(t, code, " ")
		assert.True(t, utils.ValidateBackupCodeFormat(code))
	}
}

func TestHandleStartOTPEnrollment_InvalidPassword(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), userContextKey, user)

	input := &OTPStartEnrollmentInput{}
	input.Body.Password = "wrongpassword"

	_, err = server.handleStartOTPEnrollment(ctx, input)
	require.Error(t, err)
	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 401, humaErr.Status)
}

func TestHandleCompleteOTPEnrollment_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), userContextKey, user)

	startInput := &OTPStartEnrollmentInput{}
	startInput.Body.Password = password

	startResp, err := server.handleStartOTPEnrollment(ctx, startInput)
	require.NoError(t, err)
	secret := startResp.Body.Code

	validCode, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	completeInput := &OTPCompleteEnrollmentInput{}
	completeInput.Code = validCode

	resp, err := server.handleCompleteOTPEnrollment(ctx, completeInput)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.Status)

	updatedUser, err := db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, secret, updatedUser.OTPSecret)

	backupCodes, err := db.GetBackupCodesByUserId(context.Background(), user.Id)
	require.NoError(t, err)
	assert.Len(t, backupCodes, 10)

	for _, bc := range backupCodes {
		assert.False(t, bc.Used)
	}
}

func TestHandleCompleteOTPEnrollment_InvalidCode(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), userContextKey, user)

	startInput := &OTPStartEnrollmentInput{}
	startInput.Body.Password = password

	_, err = server.handleStartOTPEnrollment(ctx, startInput)
	require.NoError(t, err)

	completeInput := &OTPCompleteEnrollmentInput{}
	completeInput.Code = "123456"

	_, err = server.handleCompleteOTPEnrollment(ctx, completeInput)
	require.Error(t, err)
	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 400, humaErr.Status)
}

func TestHandleLoginToken_WithTOTP_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{
		Name:      "Test User",
		Email:     "test@example.com",
		Role:      models.WRITE,
		OTPSecret: "JBSWY3DPEHPK3PXP",
	}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	validCode, err := totp.GenerateCode(user.OTPSecret, time.Now())
	require.NoError(t, err)

	input := &LoginInput{}
	input.Body.Email = user.Email
	input.Body.Password = password
	input.Body.OTP = validCode

	resp, err := server.handleLoginToken(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Body.Token)

	token, err := jwt.Parse(resp.Body.Token, func(token *jwt.Token) (any, error) {
		return server.jwtSecret, nil
	})
	require.NoError(t, err)
	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, user.Email, claims["email"])
}

func TestHandleLoginToken_WithBackupCode_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{
		Name:      "Test User",
		Email:     "test@example.com",
		Role:      models.WRITE,
		OTPSecret: "JBSWY3DPEHPK3PXP",
	}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	backupCodes, err := utils.GenerateBackupCodes(10)
	require.NoError(t, err)

	dbBackupCodes := make([]*models.BackupCode, len(backupCodes))
	for i, code := range backupCodes {
		dbBackupCodes[i] = &models.BackupCode{
			UserId: user.Id,
			Code:   code,
			Used:   false,
		}
	}
	err = db.CreateBackupCodes(context.Background(), dbBackupCodes)
	require.NoError(t, err)

	testCode := backupCodes[0]
	input := &LoginInput{}
	input.Body.Email = user.Email
	input.Body.Password = password
	input.Body.OTP = testCode

	resp, err := server.handleLoginToken(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Body.Token)

	usedBackupCode, err := db.GetBackupCodeByCode(context.Background(), testCode)
	require.NoError(t, err)
	require.NotNil(t, usedBackupCode)
	assert.True(t, usedBackupCode.Used)
}

func TestHandleLoginToken_WithFormattedBackupCode_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{
		Name:      "Test User",
		Email:     "test@example.com",
		Role:      models.WRITE,
		OTPSecret: "JBSWY3DPEHPK3PXP",
	}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	backupCodes, err := utils.GenerateBackupCodes(10)
	require.NoError(t, err)

	dbBackupCodes := make([]*models.BackupCode, len(backupCodes))
	for i, code := range backupCodes {
		dbBackupCodes[i] = &models.BackupCode{
			UserId: user.Id,
			Code:   code,
			Used:   false,
		}
	}
	err = db.CreateBackupCodes(context.Background(), dbBackupCodes)
	require.NoError(t, err)

	testCode := backupCodes[1]
	formattedCode := utils.FormatBackupCode(testCode)
	input := &LoginInput{}
	input.Body.Email = user.Email
	input.Body.Password = password
	input.Body.OTP = formattedCode

	resp, err := server.handleLoginToken(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Body.Token)

	usedBackupCode, err := db.GetBackupCodeByCode(context.Background(), testCode)
	require.NoError(t, err)
	require.NotNil(t, usedBackupCode)
	assert.True(t, usedBackupCode.Used)
}

func TestHandleLoginToken_WithUsedBackupCode_Failure(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{
		Name:      "Test User",
		Email:     "test@example.com",
		Role:      models.WRITE,
		OTPSecret: "JBSWY3DPEHPK3PXP",
	}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	backupCodes, err := utils.GenerateBackupCodes(1)
	require.NoError(t, err)

	backupCode := &models.BackupCode{
		UserId: user.Id,
		Code:   backupCodes[0],
		Used:   true,
	}
	err = db.CreateBackupCode(context.Background(), backupCode)
	require.NoError(t, err)

	input := &LoginInput{}
	input.Body.Email = user.Email
	input.Body.Password = password
	input.Body.OTP = backupCodes[0]

	_, err = server.handleLoginToken(context.Background(), input)
	require.Error(t, err)
	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 401, humaErr.Status)
}

func TestHandleLoginToken_WithInvalidBackupCode_Failure(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{
		Name:      "Test User",
		Email:     "test@example.com",
		Role:      models.WRITE,
		OTPSecret: "JBSWY3DPEHPK3PXP",
	}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	input := &LoginInput{}
	input.Body.Email = user.Email
	input.Body.Password = password
	input.Body.OTP = "INVALID123"

	_, err = server.handleLoginToken(context.Background(), input)
	require.Error(t, err)
	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 401, humaErr.Status)
}

func TestHandleLoginToken_WithOTPRequired_Failure(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{
		Name:      "Test User",
		Email:     "test@example.com",
		Role:      models.WRITE,
		OTPSecret: "JBSWY3DPEHPK3PXP",
	}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	input := &LoginInput{}
	input.Body.Email = user.Email
	input.Body.Password = password

	_, err = server.handleLoginToken(context.Background(), input)
	require.Error(t, err)
	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 400, humaErr.Status)
}

func TestHandleRemoveOTP_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{
		Name:      "Test User",
		Email:     "test@example.com",
		Role:      models.WRITE,
		OTPSecret: "JBSWY3DPEHPK3PXP",
	}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	backupCodes, err := utils.GenerateBackupCodes(5)
	require.NoError(t, err)

	dbBackupCodes := make([]*models.BackupCode, len(backupCodes))
	for i, code := range backupCodes {
		dbBackupCodes[i] = &models.BackupCode{
			UserId: user.Id,
			Code:   code,
			Used:   false,
		}
	}
	err = db.CreateBackupCodes(context.Background(), dbBackupCodes)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), userContextKey, user)

	input := &OTPRemoveInput{}

	resp, err := server.handleRemoveOTP(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.Status)

	updatedUser, err := db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Empty(t, updatedUser.OTPSecret)

	remainingBackupCodes, err := db.GetBackupCodesByUserId(context.Background(), user.Id)
	require.NoError(t, err)
	assert.Len(t, remainingBackupCodes, 0)
}

func TestHandleRemoveOTP_AdminRemovesOtherUser(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	adminPassword := "admin123"
	admin := &models.User{
		Name:  "Admin User",
		Email: "admin@example.com",
		Role:  models.ADMIN,
	}
	hash, err := utils.HashPassword(adminPassword)
	require.NoError(t, err)
	admin.Hash = hash
	err = db.CreateUser(context.Background(), admin)
	require.NoError(t, err)

	userPassword := "user123"
	user := &models.User{
		Name:      "Regular User",
		Email:     "user@example.com",
		Role:      models.WRITE,
		OTPSecret: "JBSWY3DPEHPK3PXP",
	}
	hash, err = utils.HashPassword(userPassword)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), userContextKey, admin)

	input := &OTPRemoveInput{}
	input.Email = user.Email

	resp, err := server.handleRemoveOTP(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.Status)

	updatedUser, err := db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Empty(t, updatedUser.OTPSecret)
}

func TestHandleRemoveOTP_NonAdminTriesToRemoveOtherUser(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	userPassword := "user123"
	user := &models.User{
		Name:  "Regular User",
		Email: "user@example.com",
		Role:  models.WRITE,
	}
	hash, err := utils.HashPassword(userPassword)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	otherUser := &models.User{
		Name:      "Other User",
		Email:     "other@example.com",
		Role:      models.WRITE,
		OTPSecret: "JBSWY3DPEHPK3PXP",
	}
	hash, err = utils.HashPassword("other123")
	require.NoError(t, err)
	otherUser.Hash = hash
	err = db.CreateUser(context.Background(), otherUser)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), userContextKey, user)

	input := &OTPRemoveInput{}
	input.Email = otherUser.Email

	_, err = server.handleRemoveOTP(ctx, input)
	require.Error(t, err)
	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}
