package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"strings"
	"time"
	"wikilite/pkg/models"
	"wikilite/pkg/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jellydator/ttlcache/v3"
	"github.com/pquerna/otp/totp"
)

const (
	// CookieName is the name of the session cookie.
	CookieName = "wiki_session"
	// SessionDuration is the duration of a user session.
	SessionDuration = 10 * time.Hour
)

// LoginInput represents the input for a user login request.
type LoginInput struct {
	Body struct {
		Email    string `format:"email"  json:"email"    required:"true"`
		Password string `json:"password" required:"true"`
		OTP      string `json:"otp" required:"false"`
	}
}

// OTPStartEnrollmentInput represents the input for an OTP enrollment request.
type OTPStartEnrollmentInput struct {
	Body struct {
		Password string `json:"password" required:"true"`
	}
}

// OTPCompleteEnrollmentInput represents the input for an OTP enrollment completion request.
type OTPCompleteEnrollmentInput struct {
	Code string `path:"code"`
}

// OTPRemoveInput represents the input for an OTP enrollment removal request.
type OTPRemoveInput struct {
	Email string `query:"email"`
}

// AuthOutput represents the output of an authentication request.
type AuthOutput struct {
	Cookies []string `header:"Set-Cookie"`
}

// AuthTokenOutput represents the output of a token creation request.
type AuthTokenOutput struct {
	Body struct {
		Type      string `json:"type"`
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expiresAt"`
	}
}

// OTPStartEnrollmentOutput represents the output of an OTP enrollment request.
type OTPStartEnrollmentOutput struct {
	Body struct {
		Code        string   `json:"secret"`
		QRCode      string   `json:"qrcode"`
		Issuer      string   `json:"issuer"`
		BackupCodes []string `json:"backupCodes"`
	}
}

// BackupCodesOutput represents the output of a backup codes request.
type BackupCodesOutput struct {
	Body struct {
		Codes []string `json:"codes"`
	}
}

// registerAuthRoutes registers the authentication routes with the API.
func (s *Server) registerAuthRoutes() {
	huma.Register(s.api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/api/login",
		Summary:     "User Login",
		Description: "Authenticate a local user.",
		Tags:        []string{"Auth"},
	}, s.handleLogin)

	huma.Register(s.api, huma.Operation{
		OperationID: "create-token",
		Method:      http.MethodPost,
		Path:        "/api/login/token",
		Summary:     "Create Token",
		Description: "Authenticate a local user. Returns a JWT token",
		Tags:        []string{"Auth"},
	}, s.handleLoginToken)

	huma.Register(s.api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        "/api/logout",
		Summary:     "User Logout",
		Description: "Clears the session.",
		Tags:        []string{"Auth"},
	}, s.handleLogout)

	huma.Register(s.api, huma.Operation{
		OperationID: "start-otp-enrollment",
		Method:      http.MethodPost,
		Path:        "/api/otp",
		Summary:     "Start OTP Enrollment",
		Description: "",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleStartOTPEnrollment)

	huma.Register(s.api, huma.Operation{
		OperationID: "complete-otp-enrollment",
		Method:      http.MethodPost,
		Path:        "/api/otp/{code}",
		Summary:     "Complete OTP Enrollment",
		Description: "",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleCompleteOTPEnrollment)

	huma.Register(s.api, huma.Operation{
		OperationID: "remove-otp",
		Method:      http.MethodDelete,
		Path:        "/api/otp",
		Summary:     "Remove OTP Enrollment",
		Description: "",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleRemoveOTP)
}

// createUserToken creates a new JWT token for a user.
func (s *Server) createUserToken(ctx context.Context, input *LoginInput) (string, error) {
	user, err := s.db.GetUserByEmail(ctx, input.Body.Email)
	if err != nil {
		return "", huma.Error500InternalServerError("Database error", err)
	}

	if user == nil {
		return "", huma.Error401Unauthorized("Invalid email or password")
	}

	if user.Disabled {
		return "", huma.Error403Forbidden("Account is disabled")
	}

	if user.IsExternal {
		return "", huma.Error400BadRequest("External users must login via their identity provider")
	}

	if !utils.CheckPassword(input.Body.Password, user.Hash) {
		return "", huma.Error401Unauthorized("Invalid email or password")
	}

	if user.OTPSecret != "" && input.Body.OTP == "" {
		return "", huma.Error400BadRequest("OTP code required")
	}

	if user.OTPSecret != "" && input.Body.OTP != "" {
		err = s.validateOTP(ctx, input.Body.OTP, user.OTPSecret, user.Id)
		if err != nil {
			return "", err
		}
	}

	claims := jwt.MapClaims{
		"sub":   fmt.Sprintf("%d", user.Id),
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
		"iss":   s.LocalIssuer,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(SessionDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.jwtSecret)
}

// validateOTP validates either a TOTP code or backup code for a user.
func (s *Server) validateOTP(ctx context.Context, otpCode, otpSecret string, userID int) error {
	if totp.Validate(otpCode, otpSecret) {
		return nil
	}

	cleanCode := strings.ReplaceAll(otpCode, " ", "")
	if !utils.ValidateBackupCodeFormat(cleanCode) {
		return huma.Error401Unauthorized("Invalid OTP code")
	}

	backupCode, err := s.db.GetBackupCodeByCode(ctx, cleanCode)
	if err != nil {
		return huma.Error500InternalServerError("Database error", err)
	}

	if backupCode == nil || backupCode.UserId != userID || backupCode.Used {
		return huma.Error401Unauthorized("Invalid backup code")
	}

	err = s.db.UseBackupCode(ctx, backupCode)
	if err != nil {
		return huma.Error500InternalServerError("Failed to use backup code", err)
	}

	return nil
}

// handleLogin handles a user login request.
func (s *Server) handleLogin(ctx context.Context, input *LoginInput) (*AuthOutput, error) {
	signedToken, err := s.createUserToken(ctx, input)
	if err != nil {
		return nil, err
	}

	cookie := http.Cookie{
		Name:     CookieName,
		Value:    signedToken,
		Path:     "/",
		Expires:  time.Now().Add(SessionDuration),
		HttpOnly: true,
		Secure:   !s.insecureCookies,
		SameSite: http.SameSiteStrictMode,
	}

	resp := &AuthOutput{}
	resp.Cookies = []string{cookie.String()}

	return resp, nil
}

// handleLoginToken handles a request to create a JWT token.
func (s *Server) handleLoginToken(
	ctx context.Context,
	input *LoginInput,
) (*AuthTokenOutput, error) {
	signedToken, err := s.createUserToken(ctx, input)
	if err != nil {
		return nil, err
	}

	resp := &AuthTokenOutput{}
	resp.Body.Token = signedToken
	resp.Body.ExpiresAt = time.Now().Add(SessionDuration).Unix()
	resp.Body.Type = "Bearer"

	return resp, nil
}

// handleLogout handles a user logout request.
func (s *Server) handleLogout(_ context.Context, _ *struct{}) (*AuthOutput, error) {
	cookie := http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		MaxAge:   -1,
	}

	resp := &AuthOutput{}
	resp.Cookies = []string{cookie.String()}

	return resp, nil
}

// handleStartOTPEnrollment handles a request to enroll an OTP secret.
func (s *Server) handleStartOTPEnrollment(
	ctx context.Context,
	input *OTPStartEnrollmentInput,
) (*OTPStartEnrollmentOutput, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error401Unauthorized("User not found in context")
	}

	if !utils.CheckPassword(input.Body.Password, user.Hash) {
		return nil, huma.Error401Unauthorized("Invalid password")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.WikiName,
		AccountName: user.Email,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to generate OTP secret", err)
	}

	s.otpCache.Set(user.Email, key.Secret(), ttlcache.DefaultTTL)

	backupCodes, err := utils.GenerateBackupCodes(10)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to generate backup codes", err)
	}

	backupCacheKey := user.Email + "_backup_codes"
	backupCodesJSON, err := json.Marshal(backupCodes)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to encode backup codes", err)
	}
	s.otpCache.Set(backupCacheKey, string(backupCodesJSON), ttlcache.DefaultTTL)

	formattedCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		formattedCodes[i] = utils.FormatBackupCode(code)
	}

	qrCodeImage, err := key.Image(256, 256)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to generate QR code", err)
	}

	var buf bytes.Buffer
	err = png.Encode(&buf, qrCodeImage)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to encode QR code", err)
	}

	qrCodeBase64 := fmt.Sprintf(
		"data:image/png;base64,%s",
		base64.StdEncoding.EncodeToString(buf.Bytes()),
	)

	resp := &OTPStartEnrollmentOutput{}
	resp.Body.Code = key.Secret()
	resp.Body.QRCode = qrCodeBase64
	resp.Body.Issuer = s.WikiName
	resp.Body.BackupCodes = formattedCodes

	return resp, nil
}

// handleCompleteOTPEnrollment handles a request to complete an OTP enrollment.
func (s *Server) handleCompleteOTPEnrollment(
	ctx context.Context,
	input *OTPCompleteEnrollmentInput,
) (*struct{ Status int }, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error401Unauthorized("User not found in context")
	}

	cachedSecret := s.otpCache.Get(user.Email)
	if cachedSecret == nil {
		return nil, huma.Error400BadRequest("OTP enrollment not found or expired")
	}

	valid := totp.Validate(input.Code, cachedSecret.Value())
	if !valid {
		return nil, huma.Error400BadRequest("Invalid OTP code")
	}

	user.OTPSecret = cachedSecret.Value()
	err := s.db.UpdateUser(ctx, user, "otp_secret")
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to save OTP secret", err)
	}

	backupCacheKey := user.Email + "_backup_codes"
	cachedBackupCodes := s.otpCache.Get(backupCacheKey)

	if cachedBackupCodes != nil {
		var backupCodes []string
		err = json.Unmarshal([]byte(cachedBackupCodes.Value()), &backupCodes)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to decode backup codes", err)
		}

		err = s.db.DeleteBackupCodesByUserId(ctx, user.Id)
		if err != nil {
			return nil, huma.Error500InternalServerError(
				"Failed to delete existing backup codes",
				err,
			)
		}

		dbBackupCodes := make([]*models.BackupCode, len(backupCodes))
		for i, code := range backupCodes {
			dbBackupCodes[i] = &models.BackupCode{
				UserId: user.Id,
				Code:   code,
				Used:   false,
			}
		}

		err = s.db.CreateBackupCodes(ctx, dbBackupCodes)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to save backup codes", err)
		}

		s.otpCache.Delete(backupCacheKey)
	}

	s.otpCache.Delete(user.Email)

	resp := &struct{ Status int }{}
	resp.Status = 200

	return resp, nil
}

// handleRemoveOTP handles a request to remove an OTP enrollment.
func (s *Server) handleRemoveOTP(
	ctx context.Context,
	input *OTPRemoveInput,
) (*struct{ Status int }, error) {
	reqUser := getUserFromContext(ctx)
	if reqUser == nil {
		return nil, huma.Error401Unauthorized("User not found in context")
	}

	var targetUser *models.User
	var err error

	if input.Email != "" {
		if reqUser.Role != models.ADMIN {
			return nil, huma.Error403Forbidden("Only admins can remove OTP for other users")
		}
		targetUser, err = s.db.GetUserByEmail(ctx, input.Email)
		if err != nil {
			return nil, huma.Error500InternalServerError("Database error", err)
		}
		if targetUser == nil {
			return nil, huma.Error404NotFound("User not found")
		}
	} else {
		targetUser = reqUser
	}

	if targetUser.OTPSecret == "" {
		return nil, huma.Error400BadRequest("User does not have OTP enabled")
	}

	targetUser.OTPSecret = ""
	err = s.db.UpdateUser(ctx, targetUser, "otp_secret")
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to remove OTP secret", err)
	}

	err = s.db.DeleteBackupCodesByUserId(ctx, targetUser.Id)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete backup codes", err)
	}

	resp := &struct{ Status int }{}
	resp.Status = 200

	return resp, nil
}
