package api

import (
	"context"
	"errors"
	"testing"
	"wikilite/pkg/models"
	"wikilite/pkg/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/golang-jwt/jwt/v5"
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
