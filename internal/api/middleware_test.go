package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"wikilite/pkg/models"
	"wikilite/pkg/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_Success(t *testing.T) {
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

	loginInput := &LoginInput{}
	loginInput.Body.Email = user.Email
	loginInput.Body.Password = password
	tokenResp, err := server.handleLoginToken(context.Background(), loginInput)
	require.NoError(t, err)

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.authMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenResp.Body.Token)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, handlerUser)
	assert.Equal(t, user.Email, handlerUser.Email)
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.authMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Nil(t, handlerUser)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.authMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Nil(t, handlerUser)
}

func TestStrictAuthMiddleware_InvalidToken(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.strictAuthMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Nil(t, handlerUser)
	assert.Contains(t, rr.Body.String(), "Invalid or expired token")
}

func TestStrictAuthMiddleware_NoToken(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.strictAuthMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Nil(t, handlerUser)
	assert.Contains(t, rr.Body.String(), "Authentication required")
}

func TestStrictAuthMiddleware_InvalidTokenFormat(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.strictAuthMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer short")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Nil(t, handlerUser)
	assert.Contains(t, rr.Body.String(), "Invalid token format")
}

func TestStrictAuthMiddleware_MalformedAuthorizationHeader(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.strictAuthMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Nil(t, handlerUser)
	assert.Contains(t, rr.Body.String(), "Authentication required")
}

func TestAuthMiddleware_IDTokenOnly(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.authMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-ID-Token", "fake-id-token")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Nil(t, handlerUser)
}

func TestAuthMiddleware_IDTokenWithValidAccessToken(t *testing.T) {
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

	loginInput := &LoginInput{}
	loginInput.Body.Email = user.Email
	loginInput.Body.Password = password
	tokenResp, err := server.handleLoginToken(context.Background(), loginInput)
	require.NoError(t, err)

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.authMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-ID-Token", "fake-id-token")
	req.Header.Set("Authorization", "Bearer "+tokenResp.Body.Token)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Nil(t, handlerUser)
}

func TestAuthMiddleware_IDTokenWithInvalidAccessToken(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.authMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-ID-Token", "fake-id-token")
	req.Header.Set("Authorization", "Bearer invalid-access-token")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Nil(t, handlerUser)
}

func TestStrictAuthMiddleware_IDTokenOnly(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.strictAuthMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-ID-Token", "fake-id-token")
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Nil(t, handlerUser)
	assert.Contains(t, rr.Body.String(), "Authentication required (Access Token missing)")
}

func TestStrictAuthMiddleware_IDTokenWithValidAccessToken(t *testing.T) {
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

	loginInput := &LoginInput{}
	loginInput.Body.Email = user.Email
	loginInput.Body.Password = password
	tokenResp, err := server.handleLoginToken(context.Background(), loginInput)
	require.NoError(t, err)

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.strictAuthMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-ID-Token", "fake-id-token")
	req.Header.Set("Authorization", "Bearer "+tokenResp.Body.Token)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Nil(t, handlerUser)
	assert.Contains(t, rr.Body.String(), "Invalid or expired ID token")
}

func TestAuthMiddleware_ExternalUserCreation(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	claims := jwt.MapClaims{
		"email": "external@example.com",
		"name":  "External User",
		"sub":   "12345",
		"iss":   "https://example.com/",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	server.jwtSecret = []byte("test-secret")
	server.externalIssuer = "https://example.com/"

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.authMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, handlerUser)
	assert.Equal(t, "external@example.com", handlerUser.Email)
	assert.Equal(t, "External User", handlerUser.Name)
	assert.Equal(t, models.READ, handlerUser.Role)
	assert.True(t, handlerUser.IsExternal)
}

func TestAuthMiddleware_ExternalUserDisabled(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{
		Name:       "Disabled User",
		Email:      "disabled@example.com",
		Role:       models.READ,
		IsExternal: true,
		Disabled:   true,
	}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"email": "disabled@example.com",
		"name":  "Disabled User",
		"sub":   "12345",
		"iss":   "https://example.com/",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	server.jwtSecret = []byte("test-secret")
	server.externalIssuer = "https://example.com/"

	var handlerUser *models.User
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerUser = getUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := server.authMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Nil(t, handlerUser)
}

func TestExtractEmailFromClaims(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name     string
		claims   jwt.MapClaims
		expected string
	}{
		{
			name: "standard email claim",
			claims: jwt.MapClaims{
				"email": "test@example.com",
			},
			expected: "test@example.com",
		},
		{
			name: "custom email claim",
			claims: jwt.MapClaims{
				"preferred_username": "test@example.com",
			},
			expected: "",
		},
		{
			name: "email claim with suffix",
			claims: jwt.MapClaims{
				"custom_email": "test@example.com",
			},
			expected: "test@example.com",
		},
		{
			name: "fallback to sub",
			claims: jwt.MapClaims{
				"sub": "user123",
			},
			expected: "user123",
		},
		{
			name:     "no email or sub",
			claims:   jwt.MapClaims{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.extractEmailFromClaims(tt.claims)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractNameFromClaims(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name     string
		claims   jwt.MapClaims
		expected string
	}{
		{
			name: "standard name claim",
			claims: jwt.MapClaims{
				"name": "John Doe",
			},
			expected: "John Doe",
		},
		{
			name: "name claim with suffix",
			claims: jwt.MapClaims{
				"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name": "John Doe",
			},
			expected: "John Doe",
		},
		{
			name:     "no name claim",
			claims:   jwt.MapClaims{},
			expected: "External User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.extractNameFromClaims(tt.claims)
			assert.Equal(t, tt.expected, result)
		})
	}
}
