//go:build ui

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUIRenderExternalIDPDisabled(t *testing.T) {
	db := newTestDB(t)

	t.Run("External IDP enabled - should show disabled page", func(t *testing.T) {
		server, err := NewServer(
			db,
			"test-secret",
			"https://example.com/.well-known/jwks.json",
			"https://example.com/",
			"email",
			"Test Wiki",
			"",
			"",
			"",
			false,
			false,
		)
		assert.NoError(t, err)

		err = server.initTemplates()
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		server.uiRenderExternalIDPDisabled(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "External Authentication Enabled")
	})
}

func TestRegisterFrontendRoutesWithExternalIDP(t *testing.T) {
	db := newTestDB(t)

	t.Run("External IDP enabled - should only register disabled route", func(t *testing.T) {
		server, err := NewServer(
			db,
			"test-secret",
			"https://example.com/.well-known/jwks.json",
			"https://example.com/",
			"email",
			"Test Wiki",
			"",
			"",
			"",
			false,
			false,
		)
		assert.NoError(t, err)

		err = server.initTemplates()
		assert.NoError(t, err)

		mux := http.NewServeMux()
		err = server.registerFrontendRoutes(mux)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/any-route", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "External Authentication Enabled")
	})

	t.Run("External IDP disabled - should register all routes", func(t *testing.T) {
		server, err := NewServer(
			db,
			"test-secret",
			"",
			"",
			"",
			"Test Wiki",
			"",
			"",
			"",
			false,
			false,
		)
		assert.NoError(t, err)

		err = server.initTemplates()
		assert.NoError(t, err)

		mux := http.NewServeMux()
		err = server.registerFrontendRoutes(mux)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/login", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Login") // Should show login page, not disabled page
	})
}
