package api

import (
	"context"
	"errors"
	"testing"
	"wikilite/pkg/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleCreateDraft_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	article, _, err := db.CreateArticleWithDraft(context.Background(), "Test Article", user.Email)
	require.NoError(t, err)

	ctx := contextWithUser(user)
	input := &ArticleSlugForDraftInput{Slug: article.Slug}
	resp, err := server.handleCreateDraft(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, article.Id, resp.Body.Draft.ArticleId)
}

func TestHandleCreateDraft_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	article, _, err := db.CreateArticleWithDraft(
		context.Background(),
		"Test Article",
		"test@example.com",
	)
	require.NoError(t, err)

	ctx := context.Background()
	input := &ArticleSlugForDraftInput{Slug: article.Slug}
	_, err = server.handleCreateDraft(ctx, input)
	require.Error(t, err)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 401, humaErr.Status)
}

func TestHandlePublishDraft_Forbidden(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user1 := &models.User{Name: "User One", Email: "user1@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user1)
	require.NoError(t, err)
	article, _, err := db.CreateArticleWithDraft(context.Background(), "Test Article", user1.Email)
	require.NoError(t, err)
	draft, err := db.CreateDraft(context.Background(), article.Id, "new content", user1.Email)
	require.NoError(t, err)

	user2 := &models.User{Name: "User Two", Email: "user2@example.com", Role: models.WRITE}
	err = db.CreateUser(context.Background(), user2)
	require.NoError(t, err)

	ctx := contextWithUser(user2)
	input := &DraftIDInput{ID: draft.Id}
	_, err = server.handlePublishDraft(ctx, input)
	require.Error(t, err)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}

func TestHandleDiscardDraft_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	article, _, err := db.CreateArticleWithDraft(context.Background(), "Test Article", user.Email)
	require.NoError(t, err)

	draft, err := db.CreateDraft(context.Background(), article.Id, "draft content", user.Email)
	require.NoError(t, err)

	ctx := contextWithUser(user)
	input := &DraftIDInput{ID: draft.Id}
	resp, err := server.handleDiscardDraft(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	_, _, err = db.GetDraftByID(context.Background(), draft.Id)
	assert.Error(t, err)
}

func TestHandleDiscardDraft_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)
	article, _, err := db.CreateArticleWithDraft(context.Background(), "Test Article", user.Email)
	require.NoError(t, err)
	draft, err := db.CreateDraft(context.Background(), article.Id, "draft content", user.Email)
	require.NoError(t, err)

	ctx := context.Background()
	input := &DraftIDInput{ID: draft.Id}
	_, err = server.handleDiscardDraft(ctx, input)
	require.Error(t, err)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 401, humaErr.Status)
}

func TestHandleDiscardDraft_Forbidden(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user1 := &models.User{Name: "User One", Email: "user1@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user1)
	require.NoError(t, err)
	article, _, err := db.CreateArticleWithDraft(context.Background(), "Test Article", user1.Email)
	require.NoError(t, err)
	draft, err := db.CreateDraft(context.Background(), article.Id, "draft content", user1.Email)
	require.NoError(t, err)

	user2 := &models.User{Name: "User Two", Email: "user2@example.com", Role: models.WRITE}
	err = db.CreateUser(context.Background(), user2)
	require.NoError(t, err)

	ctx := contextWithUser(user2)
	input := &DraftIDInput{ID: draft.Id}
	_, err = server.handleDiscardDraft(ctx, input)
	require.Error(t, err)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}
