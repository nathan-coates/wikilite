package api

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"wikilite/pkg/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/golang-jwt/jwt/v5"
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
	}
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
		Secure:   true,
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
