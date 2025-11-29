package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wikilite/pkg/models"
)

func TestCreateUser(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}

	err := db.CreateUser(ctx, user)
	require.NoError(t, err)
	assert.NotZero(t, user.Id)
	assert.NotZero(t, user.CreatedAt)
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	user1 := &models.User{
		Name:  "User One",
		Email: "duplicate@example.com",
		Role:  models.WRITE,
	}
	user2 := &models.User{
		Name:  "User Two",
		Email: "duplicate@example.com",
		Role:  models.WRITE,
	}

	err := db.CreateUser(ctx, user1)
	require.NoError(t, err)

	err = db.CreateUser(ctx, user2)
	assert.Error(t, err)
}

func TestGetUserByEmail(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	original := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.ADMIN,
	}
	err := db.CreateUser(ctx, original)
	require.NoError(t, err)

	found, err := db.GetUserByEmail(ctx, "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, original.Id, found.Id)
	assert.Equal(t, original.Name, found.Name)
	assert.Equal(t, original.Email, found.Email)
	assert.Equal(t, original.Role, found.Role)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	found, err := db.GetUserByEmail(ctx, "nonexistent@example.com")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestTypedErrors(t *testing.T) {
	assert.NotNil(t, ErrCannotEditDraft)
	assert.NotNil(t, ErrCannotDiscardDraft)
	assert.Equal(t, "unauthorized: you cannot edit this draft", ErrCannotEditDraft.Error())
	assert.Equal(t, "unauthorized: you cannot discard this draft", ErrCannotDiscardDraft.Error())
}
