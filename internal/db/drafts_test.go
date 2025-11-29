package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wikilite/pkg/models"
)

func TestCreateDraft(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	article, _, err := db.CreateArticleWithDraft(ctx, "Test Article", user.Email)
	require.NoError(t, err)

	draft, err := db.CreateDraft(ctx, article.Id, "# Updated Content", user.Email)
	require.NoError(t, err)
	assert.NotZero(t, draft.Id)
	assert.Equal(t, article.Id, draft.ArticleId)
	assert.Equal(t, user.Email, draft.CreatedBy)
	assert.NotZero(t, draft.CreatedAt)
}

func TestUpdateDraft_Success(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	article, _, err := db.CreateArticleWithDraft(ctx, "Test Article", user.Email)
	require.NoError(t, err)

	draft, err := db.CreateDraft(ctx, article.Id, "# First Update", user.Email)
	require.NoError(t, err)

	err = db.UpdateDraft(ctx, draft.Id, "# Second Update", user.Email)
	require.NoError(t, err)

	_, content, err := db.GetDraftByID(ctx, draft.Id)
	require.NoError(t, err)
	assert.Equal(t, "# Second Update", content)
}

func TestUpdateDraft_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	user1 := &models.User{
		Name:  "User One",
		Email: "user1@example.com",
		Role:  models.WRITE,
	}
	err := db.CreateUser(ctx, user1)
	require.NoError(t, err)

	user2 := &models.User{
		Name:  "User Two",
		Email: "user2@example.com",
		Role:  models.WRITE,
	}
	err = db.CreateUser(ctx, user2)
	require.NoError(t, err)

	article, _, err := db.CreateArticleWithDraft(ctx, "Test Article", user1.Email)
	require.NoError(t, err)

	draft, err := db.CreateDraft(ctx, article.Id, "# First Update", user1.Email)
	require.NoError(t, err)

	err = db.UpdateDraft(ctx, draft.Id, "# Malicious Update", user2.Email)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCannotEditDraft))
}

func TestDiscardDraft_Success(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	article, _, err := db.CreateArticleWithDraft(ctx, "Test Article", user.Email)
	require.NoError(t, err)

	draft, err := db.CreateDraft(ctx, article.Id, "# Updated Content", user.Email)
	require.NoError(t, err)

	err = db.DiscardDraft(ctx, draft.Id, user.Email)
	require.NoError(t, err)

	_, _, err = db.GetDraftByID(ctx, draft.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, sql.ErrNoRows))
}

func TestDiscardDraft_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	user1 := &models.User{
		Name:  "User One",
		Email: "user1@example.com",
		Role:  models.WRITE,
	}
	err := db.CreateUser(ctx, user1)
	require.NoError(t, err)

	user2 := &models.User{
		Name:  "User Two",
		Email: "user2@example.com",
		Role:  models.WRITE,
	}
	err = db.CreateUser(ctx, user2)
	require.NoError(t, err)

	article, _, err := db.CreateArticleWithDraft(ctx, "Test Article", user1.Email)
	require.NoError(t, err)

	draft, err := db.CreateDraft(ctx, article.Id, "# Updated Content", user1.Email)
	require.NoError(t, err)

	err = db.DiscardDraft(ctx, draft.Id, user2.Email)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCannotDiscardDraft))

	_, _, err = db.GetDraftByID(ctx, draft.Id)
	assert.NoError(t, err)
}

func TestPublishDraft_Success(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	article, _, err := db.CreateArticleWithDraft(ctx, "Test Article", user.Email)
	require.NoError(t, err)

	draft, err := db.CreateDraft(
		ctx,
		article.Id,
		"# Updated Content\n\nThis is new content.",
		user.Email,
	)
	require.NoError(t, err)

	err = db.PublishDraft(ctx, draft.Id)
	require.NoError(t, err)

	updatedArticle, err := db.GetArticleBySlug(ctx, "test-article")
	require.NoError(t, err)
	assert.Contains(t, updatedArticle.Data, "This is new content")
	assert.Equal(t, 1, updatedArticle.Version)

	_, _, err = db.GetDraftByID(ctx, draft.Id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, sql.ErrNoRows))
}

func TestGetDraftByID(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	article, _, err := db.CreateArticleWithDraft(ctx, "Test Article", user.Email)
	require.NoError(t, err)

	original, err := db.CreateDraft(ctx, article.Id, "# Updated Content", user.Email)
	require.NoError(t, err)

	found, content, err := db.GetDraftByID(ctx, original.Id)
	require.NoError(t, err)
	assert.Equal(t, original.Id, found.Id)
	assert.Equal(t, original.ArticleId, found.ArticleId)
	assert.Equal(t, "# Updated Content", content)
	assert.Equal(t, original.CreatedBy, found.CreatedBy)
}

func TestGetDraftByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	found, _, err := db.GetDraftByID(ctx, 999)
	assert.Error(t, err)
	assert.Nil(t, found)
	assert.True(t, errors.Is(err, sql.ErrNoRows))
}
