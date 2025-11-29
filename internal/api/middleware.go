package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"wikilite/pkg/models"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a private type to prevent key collisions in context.
type contextKey string

// userContextKey is the key used to store/retrieve the user from context.
const userContextKey contextKey = "user"

// responseWriter is a wrapper around http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter

	status int
}

// WriteHeader captures the status code before writing it to the response.
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// ContextMiddleware injects global dependencies (like the DB logger) into the request context.
func (s *Server) contextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := models.NewContextWithLogger(r.Context(), s.db.CreateLogEntry)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggerMiddleware logs HTTP requests to the database asynchronously.
func (s *Server) LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		level := models.LevelInfo
		if rw.status >= 400 && rw.status < 500 {
			level = models.LevelWarning
		} else if rw.status >= 500 {
			level = models.LevelError
		}

		userLog := "Anonymous"
		user := getUserFromContext(r.Context())
		if user != nil {
			userLog = user.Email
		}
		message := fmt.Sprintf("%s %s - %d", r.Method, r.URL.Path, rw.status)
		data := fmt.Sprintf("User: %s | Duration: %s | IP: %s | UserAgent: %s",
			userLog, duration, r.RemoteAddr, r.UserAgent())

		_ = s.db.CreateLogEntry(context.Background(), level, "API", message, data)
	})
}

// authMiddleware checks for a Bearer token, validates it, and sets the user in context.
// It is "soft" authentication: if no token or invalid token, it proceeds with user=nil.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return s.authMiddlewareWithOptions(next, false)
}

// strictAuthMiddleware checks for a Bearer token, validates it, and sets the user in context.
// It is "strict" authentication: if no token or invalid token, it returns 401 Unauthorized.
func (s *Server) strictAuthMiddleware(next http.Handler) http.Handler {
	return s.authMiddlewareWithOptions(next, true)
}

// authMiddlewareWithOptions implements the core authentication logic with strict/soft mode.
func (s *Server) authMiddlewareWithOptions(next http.Handler, strict bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idToken := r.Header.Get("X-ID-Token")
		tokenString := s.extractTokenFromRequest(r)

		fail := func(msg string) {
			if strict {
				w.Header().Set("WWW-Authenticate", "Bearer")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(fmt.Sprintf(`{"error":"%s"}`, msg)))
			} else {
				next.ServeHTTP(w, r)
			}
		}

		if idToken != "" {
			if tokenString == "" {
				fail("Authentication required (Access Token missing)")
				return
			}

			_, err := s.parseJWT(tokenString)
			if err != nil {
				fail("Invalid or expired access token")
				return
			}

			user, err := s.validateToken(r.Context(), idToken)
			if err != nil {
				fail("Invalid or expired ID token")
				return
			}

			if user != nil {
				ctx := context.WithValue(r.Context(), userContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		if tokenString != "" {
			if len(tokenString) < 10 || strings.Contains(tokenString, " ") {
				fail("Invalid token format")
				return
			}

			user, err := s.validateToken(r.Context(), tokenString)
			if err != nil {
				fail("Invalid or expired token")
				return
			}

			if user != nil {
				ctx := context.WithValue(r.Context(), userContextKey, user)
				r = r.WithContext(ctx)
			}
		} else if strict {
			fail("Authentication required")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// parseJWT parses and cryptographically validates a token, returning its claims.
// It checks signature, expiration, and issuer.
func (s *Server) parseJWT(tokenString string) (jwt.MapClaims, error) {
	var token *jwt.Token
	var err error

	if s.jwks != nil {
		token, err = jwt.Parse(tokenString, s.jwks.Keyfunc)
	} else {
		token, err = jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.jwtSecret, nil
		})
	}

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	if s.externalIssuer != "" {
		iss, _ := claims.GetIssuer()
		if iss != s.externalIssuer {
			return nil, fmt.Errorf("invalid issuer: expected %s, got %s", s.externalIssuer, iss)
		}
	}

	return claims, nil
}

// validateToken parses a token, validates it, and resolves the User from the DB.
func (s *Server) validateToken(ctx context.Context, tokenString string) (*models.User, error) {
	claims, err := s.parseJWT(tokenString)
	if err != nil {
		return nil, err
	}

	email := s.extractEmailFromClaims(claims)
	if email == "" {
		return nil, fmt.Errorf("email claim is empty (checked: %s)", s.jwtEmailClaim)
	}

	user, err := s.db.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if user == nil {
		return s.createExternalUser(ctx, claims, email)
	}

	if user.Disabled {
		return nil, fmt.Errorf("user account is disabled")
	}

	return user, nil
}

// extractEmailFromClaims extracts email from JWT claims using various strategies.
func (s *Server) extractEmailFromClaims(claims jwt.MapClaims) string {
	if s.jwtEmailClaim != "" {
		if v, ok := claims[s.jwtEmailClaim].(string); ok {
			return v
		}
		return ""
	}

	if v, ok := claims["email"].(string); ok {
		return v
	}

	for k, v := range claims {
		if strings.HasSuffix(k, "email") {
			if strVal, ok := v.(string); ok {
				return strVal
			}
		}
	}

	if v, ok := claims["sub"].(string); ok {
		return v
	}

	return ""
}

// createExternalUser creates a new external user from JWT claims.
func (s *Server) createExternalUser(
	ctx context.Context,
	claims jwt.MapClaims,
	email string,
) (*models.User, error) {
	name := s.extractNameFromClaims(claims)

	newUser := &models.User{
		Name:       name,
		Email:      email,
		Role:       models.READ,
		IsExternal: true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Hash:       "",
	}

	err := s.db.CreateUser(ctx, newUser)
	if err != nil {
		return nil, fmt.Errorf("failed to provision new user: %w", err)
	}

	return newUser, nil
}

// extractNameFromClaims extracts name from JWT claims using various strategies.
func (s *Server) extractNameFromClaims(claims jwt.MapClaims) string {
	if n, ok := claims["name"].(string); ok {
		return n
	}

	for k, v := range claims {
		if strings.HasSuffix(k, "/name") {
			if strVal, ok := v.(string); ok {
				return strVal
			}
		}
	}

	return "External User"
}

// extractTokenFromRequest extracts token from cookie or Authorization header.
func (s *Server) extractTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(CookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	if len(parts) == 1 && parts[0] != "" {
		return parts[0]
	}

	return ""
}

// getUserFromContext is a helper to retrieve the user safely from a request context.
func getUserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(userContextKey).(*models.User)
	if !ok {
		return nil
	}

	return user
}

// getAdminUserFromContext retrieves the user ONLY if they have the ADMIN role.
func getAdminUserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(userContextKey).(*models.User)
	if !ok || user == nil {
		return nil
	}

	if user.Role != models.ADMIN {
		return nil
	}

	return user
}
