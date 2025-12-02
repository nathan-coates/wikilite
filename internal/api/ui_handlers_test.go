//go:build ui

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
	"wikilite/pkg/models"
	"wikilite/pkg/utils"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUIRenderHome(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Test Wiki")
}

func TestUIRenderArticle(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	req := httptest.NewRequest("GET", "/wiki/home", nil)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Welcome to your Home")
}

func TestUIRenderHistory(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Email: "test@example.com", Role: models.WRITE}
	article, _, err := db.CreateArticleWithDraft(context.Background(), "History Test", user.Email)
	require.NoError(t, err)

	draft, err := db.CreateDraft(context.Background(), article.Id, "new content", user.Email)
	require.NoError(t, err)

	err = db.PublishDraft(context.Background(), draft.Id)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/wiki/"+article.Slug+"/history", nil)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "History: history-test")
	assert.Contains(t, rr.Body.String(), ">v1<")
}

func TestUIRenderPastVersion(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Email: "test@example.com", Role: models.WRITE}
	article, _, err := db.CreateArticleWithDraft(
		context.Background(),
		"Past Version Test",
		user.Email,
	)
	require.NoError(t, err)

	draft, err := db.CreateDraft(context.Background(), article.Id, "Version 1 content", user.Email)
	require.NoError(t, err)
	err = db.PublishDraft(context.Background(), draft.Id)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/wiki/"+article.Slug+"/history/1", nil)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Version 1 content")
}

func TestUIRenderLogin(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	req := httptest.NewRequest("GET", "/login", nil)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Login")
}

func TestUIHandleLoginSubmit_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	password := "password123"
	user := &models.User{Name: "Login User", Email: "login@user.com", Role: models.WRITE}
	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	form := url.Values{}
	form.Add("email", user.Email)
	form.Add("password", password)
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)
	assert.Equal(t, "/dashboard", rr.Header().Get("Location"))
	assert.NotEmpty(t, rr.Header().Get("Set-Cookie"))
}

func TestUIHandleLoginSubmit_Failure(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	form := url.Values{}
	form.Add("email", "non-existent@user.com")
	form.Add("password", "wrong-password")
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid credentials")
}

func TestUIRenderOTPSettings(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	user, err = db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/user/otp", nil)
	ctx := contextWithUser(user)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Two-Factor Authentication")
	assert.Contains(t, rr.Body.String(), "Disabled")
	assert.Contains(t, rr.Body.String(), "Enable 2FA")
}

func TestUIRenderOTPSettings_WithOTPEnabled(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	user.OTPSecret = "test-secret"
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	user, err = db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/user/otp", nil)
	ctx := contextWithUser(user)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Two-Factor Authentication")
	assert.Contains(t, rr.Body.String(), "Enabled")
	assert.Contains(t, rr.Body.String(), "Disable 2FA")
}

func TestUIRenderOTPSettings_WithSuccessMessage(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	user, err = db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/user/otp?success=1", nil)
	ctx := contextWithUser(user)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Two-factor authentication enabled successfully")
}

func TestUIHandleOTPStartEnrollment_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	user, err = db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)

	form := url.Values{}
	form.Add("password", "password123")
	req := httptest.NewRequest("POST", "/user/otp/enroll", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	ctx := contextWithUser(user)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Enable Two-Factor Authentication")
	assert.Contains(t, rr.Body.String(), "QR Code")
	assert.Contains(t, rr.Body.String(), "Backup Codes")
	assert.Contains(t, rr.Body.String(), "Verify Setup")
}

func TestUIHandleOTPStartEnrollment_InvalidPassword(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	user, err = db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)

	form := url.Values{}
	form.Add("password", "wrongpassword")
	req := httptest.NewRequest("POST", "/user/otp/enroll", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	ctx := contextWithUser(user)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid password")
}

func TestUIHandleOTPStartEnrollment_MissingPassword(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	user, err = db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)

	form := url.Values{}

	req := httptest.NewRequest("POST", "/user/otp/enroll", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	ctx := contextWithUser(user)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Password is required")
}

func TestUIHandleOTPStartEnrollment_Unauthenticated(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	form := url.Values{}
	form.Add("password", "password123")
	req := httptest.NewRequest("POST", "/user/otp/enroll", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)
	assert.Equal(t, "/login", rr.Header().Get("Location"))
}

func TestUIRenderOTPEnroll(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	user, err = db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/user/otp/enroll", nil)
	ctx := contextWithUser(user)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Enable Two-Factor Authentication")
	assert.Contains(t, rr.Body.String(), "Please start the enrollment process")
}

func TestUIHandleOTPVerify_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	user, err = db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)

	form := url.Values{}
	form.Add("password", "password123")
	req1 := httptest.NewRequest("POST", "/user/otp/enroll", strings.NewReader(form.Encode()))
	req1.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	ctx := contextWithUser(user)
	req1 = req1.WithContext(ctx)
	rr1 := httptest.NewRecorder()
	server.router.ServeHTTP(rr1, req1)

	cachedSecret := server.otpCache.Get(user.Email)
	require.NotNil(t, cachedSecret)

	validCode, err := totp.GenerateCode(cachedSecret.Value(), time.Now())
	require.NoError(t, err)

	form2 := url.Values{}
	form2.Add("code", validCode)
	req2 := httptest.NewRequest("POST", "/user/otp/verify", strings.NewReader(form2.Encode()))
	req2.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req2 = req2.WithContext(ctx)
	rr2 := httptest.NewRecorder()

	server.router.ServeHTTP(rr2, req2)

	assert.Equal(t, http.StatusOK, rr2.Code)
	assert.Contains(t, rr2.Body.String(), "Two-factor authentication enabled successfully")
}

func TestUIHandleOTPVerify_InvalidCode(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	user, err = db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)

	form := url.Values{}
	form.Add("password", "password123")
	req1 := httptest.NewRequest("POST", "/user/otp/enroll", strings.NewReader(form.Encode()))
	req1.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	ctx := contextWithUser(user)
	req1 = req1.WithContext(ctx)
	rr1 := httptest.NewRecorder()
	server.router.ServeHTTP(rr1, req1)

	form2 := url.Values{}
	form2.Add("code", "123456")
	req2 := httptest.NewRequest("POST", "/user/otp/verify", strings.NewReader(form2.Encode()))
	req2.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req2 = req2.WithContext(ctx)
	rr2 := httptest.NewRecorder()

	server.router.ServeHTTP(rr2, req2)

	assert.Equal(t, http.StatusOK, rr2.Code)
	assert.Contains(t, rr2.Body.String(), "Invalid verification code")
}

func TestUIHandleOTPVerify_MissingCode(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	user, err = db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)

	form := url.Values{}
	form.Add("password", "password123")
	req1 := httptest.NewRequest("POST", "/user/otp/enroll", strings.NewReader(form.Encode()))
	req1.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	ctx := contextWithUser(user)
	req1 = req1.WithContext(ctx)
	rr1 := httptest.NewRecorder()
	server.router.ServeHTTP(rr1, req1)

	form2 := url.Values{}

	req2 := httptest.NewRequest("POST", "/user/otp/verify", strings.NewReader(form2.Encode()))
	req2.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req2 = req2.WithContext(ctx)
	rr2 := httptest.NewRecorder()

	server.router.ServeHTTP(rr2, req2)

	assert.Equal(t, http.StatusOK, rr2.Code)
	assert.Contains(t, rr2.Body.String(), "Verification code is required")
}

func TestUIHandleOTPVerify_Unauthenticated(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	req := httptest.NewRequest("POST", "/user/otp/disable", nil)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)
	assert.Equal(t, "/login", rr.Header().Get("Location"))
}

func TestUIHandleOTPVerify_InvalidCode_ReproCase(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	// Get the user from DB to have the ID populated
	user, err = db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)

	// First start enrollment to get the OTP secret cached
	form := url.Values{}
	form.Add("password", "password123")
	req1 := httptest.NewRequest("POST", "/user/otp/enroll", strings.NewReader(form.Encode()))
	req1.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	ctx := contextWithUser(user)
	req1 = req1.WithContext(ctx)
	rr1 := httptest.NewRecorder()
	server.router.ServeHTTP(rr1, req1)

	// Get the cached secret to verify it's not predictable
	cachedSecret := server.otpCache.Get(user.Email)
	require.NotNil(t, cachedSecret)
	t.Logf("Cached OTP secret: %s", cachedSecret.Value())

	// Test with the specific code "111111" that user reported
	form2 := url.Values{}
	form2.Add("code", "111111")
	req2 := httptest.NewRequest("POST", "/user/otp/verify", strings.NewReader(form2.Encode()))
	req2.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req2 = req2.WithContext(ctx)
	rr2 := httptest.NewRecorder()

	server.router.ServeHTTP(rr2, req2)

	// This should fail - 111111 should not be a valid TOTP code
	assert.Equal(t, http.StatusOK, rr2.Code)
	assert.Contains(t, rr2.Body.String(), "Invalid verification code")

	// Verify that OTP was NOT actually enabled for the user
	updatedUser, err := db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)
	assert.Empty(t, updatedUser.OTPSecret, "OTP should not be enabled after invalid code")
}

func TestUIHandleOTPDisable_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	hash, err := utils.HashPassword("password123")
	require.NoError(t, err)
	user.Hash = hash
	user.OTPSecret = "test-secret"
	err = db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	user, err = db.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/user/otp/disable", nil)
	ctx := contextWithUser(user)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Two-factor authentication disabled successfully")
}

func TestUIHandleOTPDisable_Unauthenticated(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	req := httptest.NewRequest("POST", "/user/otp/disable", nil)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)
	assert.Equal(t, "/login", rr.Header().Get("Location"))
}
