package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"wikilite/pkg/models"
	"wikilite/pkg/utils"

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
