//go:build ui

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"wikilite/pkg/models"
	"wikilite/pkg/utils"

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
