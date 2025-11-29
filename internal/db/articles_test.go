package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wikilite/pkg/models"
)

func TestCreateArticleWithDraft(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	article, draft, err := db.CreateArticleWithDraft(ctx, "Test Article", "test@example.com")
	require.NoError(t, err)
	assert.NotZero(t, article.Id)
	assert.NotZero(t, draft.Id)
	assert.Equal(t, article.Id, draft.ArticleId)
	assert.Equal(t, user.Email, article.CreatedBy)
	assert.Equal(t, user.Email, draft.CreatedBy)
}

func TestGetArticleBySlug(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  models.WRITE,
	}
	err := db.CreateUser(ctx, user)
	require.NoError(t, err)

	original, _, err := db.CreateArticleWithDraft(ctx, "Test Article", user.Email)
	require.NoError(t, err)

	found, err := db.GetArticleBySlug(ctx, "test-article")
	require.NoError(t, err)
	assert.Equal(t, original.Id, found.Id)
	assert.Equal(t, original.Title, found.Title)
	assert.Equal(t, original.Slug, found.Slug)
	assert.Equal(t, original.Data, found.Data)
}

func TestGetArticleBySlug_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	found, err := db.GetArticleBySlug(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestGetArticlesByUser(t *testing.T) {
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

	_, _, err = db.CreateArticleWithDraft(ctx, "First Article", user1.Email)
	require.NoError(t, err)

	_, _, err = db.CreateArticleWithDraft(ctx, "Second Article", user1.Email)
	require.NoError(t, err)

	_, _, err = db.CreateArticleWithDraft(ctx, "Third Article", user2.Email)
	require.NoError(t, err)

	articles, err := db.GetArticlesByUser(ctx, user1.Email)
	require.NoError(t, err)
	assert.Len(t, articles, 2)

	articles, err = db.GetArticlesByUser(ctx, user2.Email)
	require.NoError(t, err)
	assert.Len(t, articles, 1)
	assert.Equal(t, "third-article", articles[0].Slug)
}
