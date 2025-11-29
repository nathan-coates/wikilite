package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"wikilite/pkg/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleCreateArticle_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Email: "test@example.com", Role: models.WRITE}
	ctx := contextWithUser(user)

	input := &CreateArticleInput{}
	input.Body.Title = "My New Article"

	resp, err := server.handleCreateArticle(ctx, input)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "my-new-article", resp.Body.ArticleSlug)

	article, err := db.GetArticleBySlug(context.Background(), "my-new-article")
	require.NoError(t, err)
	require.NotNil(t, article)
	assert.Equal(t, "My New Article", article.Title)
	assert.Equal(t, user.Email, article.CreatedBy)

	draft, _, err := db.GetDraftByID(context.Background(), resp.Body.DraftID)
	require.NoError(t, err)
	require.NotNil(t, draft)
	assert.Equal(t, article.Id, draft.ArticleId)
}

func TestHandleCreateArticle_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	ctx := context.Background()

	input := &CreateArticleInput{}
	input.Body.Title = "My New Article"

	resp, err := server.handleCreateArticle(ctx, input)

	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok, "Error should be a huma.ErrorModel")
	assert.Equal(t, 401, humaErr.Status)
}

func TestHandleGetArticleJSON_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	ctx := context.Background()

	input := &ArticleSlugInput{Slug: "home"}

	resp, err := server.handleGetArticleJSON(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "Home", resp.Body.PublicArticle.Title)
	assert.Nil(t, resp.Body.PublicArticle.Author, "Author should be nil for non-admin users")
}

func TestHandleGetArticleJSON_NotFound(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	ctx := context.Background()

	input := &ArticleSlugInput{Slug: "non-existent-slug"}

	resp, err := server.handleGetArticleJSON(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 404, humaErr.Status)
}

func TestHandleGetArticleJSON_AdminView(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	admin := &models.User{Email: "admin@test.com", Role: models.ADMIN}
	ctx := contextWithUser(admin)

	input := &ArticleSlugInput{Slug: "home"}

	resp, err := server.handleGetArticleJSON(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "Home", resp.Body.PublicArticle.Title)
	require.NotNil(t, resp.Body.PublicArticle.Author)

	assert.Equal(t, "1", *resp.Body.PublicArticle.Author)
}

func TestHandleGetArticleContent_HTML(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	ctx := context.Background()
	input := &ArticleContentInput{Slug: "home", Format: "html"}

	resp, err := server.handleGetArticleContent(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	op := &huma.Operation{
		OperationID: "get-article-content",
		Method:      http.MethodGet,
		Path:        "/api/articles/{slug}/content",
	}
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/api/articles/home/content", nil)
	hctx := humatest.NewContext(op, r, w)

	resp.Body(hctx)

	body := w.Body.String()

	assert.Contains(t, body, "Welcome to your Home")
}

func TestHandleGetArticleContent_Markdown(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	ctx := context.Background()
	input := &ArticleContentInput{Slug: "home", Format: "md"}

	resp, err := server.handleGetArticleContent(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	op := &huma.Operation{
		OperationID: "get-article-content",
		Method:      http.MethodGet,
		Path:        "/api/articles/{slug}/content",
	}
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/api/articles/home/content", nil)
	hctx := humatest.NewContext(op, r, w)

	resp.Body(hctx)

	body := w.Body.String()

	assert.Contains(t, body, "# Welcome to your Home")
}

func TestHandleGetArticleContent_NotFound(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	ctx := context.Background()
	input := &ArticleContentInput{Slug: "non-existent", Format: "html"}

	resp, err := server.handleGetArticleContent(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 404, humaErr.Status)
}

func TestHandleGetArticleVersion_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Email: "test@example.com", Role: models.WRITE}
	article, _, err := db.CreateArticleWithDraft(
		context.Background(),
		"Versioned Article",
		user.Email,
	)
	require.NoError(t, err)

	draft, err := db.CreateDraft(context.Background(), article.Id, "Version 1 content", user.Email)
	require.NoError(t, err)
	err = db.PublishDraft(context.Background(), draft.Id)
	require.NoError(t, err)

	ctx := context.Background()
	input := &ArticleVersionInput{Slug: article.Slug, Version: 1, Format: "html"}
	resp, err := server.handleGetArticleVersion(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	op := &huma.Operation{
		OperationID: "get-article-version",
		Method:      http.MethodGet,
		Path:        "/api/articles/{slug}/versions/{version}",
	}
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/api/articles/versioned-article/versions/1", nil)
	hctx := humatest.NewContext(op, r, w)

	resp.Body(hctx)

	body := w.Body.String()
	assert.Contains(t, body, "Version 1 content")
}

func TestHandleGetArticleVersion_InvalidVersion(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	ctx := context.Background()
	input := &ArticleVersionInput{Slug: "home", Version: 99, Format: "html"}

	resp, err := server.handleGetArticleVersion(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	humaErr, ok := err.(*huma.ErrorModel)
	require.True(t, ok)
	assert.Equal(t, 404, humaErr.Status)
}

func TestHandleGetOrphans_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Email: "test@example.com", Role: models.WRITE}

	articleA, _, err := db.CreateArticleWithDraft(context.Background(), "Article A", user.Email)
	require.NoError(t, err)

	articleB, _, err := db.CreateArticleWithDraft(context.Background(), "Article B", user.Email)
	require.NoError(t, err)

	draft, err := db.CreateDraft(
		context.Background(),
		articleA.Id,
		"Link to B: [link](/"+articleB.Slug+")",
		user.Email,
	)
	require.NoError(t, err)

	err = db.PublishDraft(context.Background(), draft.Id)
	require.NoError(t, err)

	admin := &models.User{Email: "admin@test.com", Role: models.ADMIN}
	ctx := contextWithUser(admin)

	resp, err := server.handleGetOrphans(ctx, &struct{}{})
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Len(t, resp.Body.Articles, 1)
	assert.Equal(t, "Article A", resp.Body.Articles[0].Title)
}

func TestHandleGetOrphans_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Email: "test@example.com", Role: models.WRITE}
	ctx := contextWithUser(user)

	resp, err := server.handleGetOrphans(ctx, &struct{}{})
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}

func TestHandleGetArticlesByUser_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Name: "Test User", Email: "test@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user)
	require.NoError(t, err)

	_, _, err = db.CreateArticleWithDraft(context.Background(), "User Article", user.Email)
	require.NoError(t, err)

	ctx := contextWithUser(user)

	input := &ArticleListInput{}
	resp, err := server.handleGetArticlesByUser(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Len(t, resp.Body.Articles, 1)
	assert.Equal(t, "User Article", resp.Body.Articles[0].Title)
}

func TestHandleGetArticlesByUser_Admin_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user1 := &models.User{Name: "User One", Email: "user1@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user1)
	require.NoError(t, err)
	_, _, err = db.CreateArticleWithDraft(context.Background(), "User One Article", user1.Email)
	require.NoError(t, err)

	user2 := &models.User{Name: "User Two", Email: "user2@example.com", Role: models.WRITE}
	err = db.CreateUser(context.Background(), user2)
	require.NoError(t, err)
	_, _, err = db.CreateArticleWithDraft(context.Background(), "User Two Article", user2.Email)
	require.NoError(t, err)

	admin := &models.User{Email: "admin@test.com", Role: models.ADMIN}
	ctx := contextWithUser(admin)

	input := &ArticleListInput{Email: user1.Email}
	resp, err := server.handleGetArticlesByUser(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Len(t, resp.Body.Articles, 1)
	assert.Equal(t, "User One Article", resp.Body.Articles[0].Title)
}

func TestHandleGetArticlesByUser_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	ctx := context.Background() // No user in context

	input := &ArticleListInput{}
	resp, err := server.handleGetArticlesByUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	var humaErr *huma.ErrorModel
	ok := errors.As(err, &humaErr)
	require.True(t, ok)
	assert.Equal(t, 401, humaErr.Status)
}

func TestHandleGetArticlesByUser_Forbidden(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user1 := &models.User{Name: "User One", Email: "user1@example.com", Role: models.WRITE}
	err := db.CreateUser(context.Background(), user1)
	require.NoError(t, err)

	user2 := &models.User{Name: "User Two", Email: "user2@example.com", Role: models.WRITE}
	err = db.CreateUser(context.Background(), user2)
	require.NoError(t, err)

	ctx := contextWithUser(user1)

	input := &ArticleListInput{Email: user2.Email}
	resp, err := server.handleGetArticlesByUser(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	humaErr, ok := err.(*huma.ErrorModel)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}

func TestHandleGetArticleHistory_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Email: "test@example.com", Role: models.WRITE}
	article, _, err := db.CreateArticleWithDraft(
		context.Background(),
		"History Article",
		user.Email,
	)
	require.NoError(t, err)

	draft, err := db.CreateDraft(context.Background(), article.Id, "History content", user.Email)
	require.NoError(t, err)
	err = db.PublishDraft(context.Background(), draft.Id)
	require.NoError(t, err)

	ctx := context.Background()
	input := &ArticleSlugInput{Slug: article.Slug}
	resp, err := server.handleGetArticleHistory(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Len(t, resp.Body.History, 1)
	assert.Equal(t, 1, resp.Body.History[0].Version)
}

func TestHandleGetArticleHistory_NotFound(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	ctx := context.Background()
	input := &ArticleSlugInput{Slug: "non-existent"}

	resp, err := server.handleGetArticleHistory(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	humaErr, ok := err.(*huma.ErrorModel)
	require.True(t, ok)
	assert.Equal(t, 404, humaErr.Status)
}

func TestHandleGetArticles_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Email: "test@example.com", Role: models.WRITE}
	_, _, err := db.CreateArticleWithDraft(context.Background(), "Article 1", user.Email)
	require.NoError(t, err)
	_, _, err = db.CreateArticleWithDraft(context.Background(), "Article 2", user.Email)
	require.NoError(t, err)

	ctx := context.Background()
	input := &ArticlePaginationInput{Page: 1, Limit: 2}
	resp, err := server.handleGetArticles(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.EqualValues(t, 3, resp.Body.Total)
	assert.Len(t, resp.Body.Articles, 2)
	assert.Equal(t, 1, resp.Body.Page)
	assert.Equal(t, 2, resp.Body.Limit)
}

func TestHandleGetArticles_Pagination(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Email: "test@example.com", Role: models.WRITE}
	for i := range 10 {
		_, _, err := db.CreateArticleWithDraft(
			context.Background(),
			fmt.Sprintf("Article %d", i),
			user.Email,
		)
		require.NoError(t, err)
	}

	ctx := context.Background()
	input := &ArticlePaginationInput{Page: 2, Limit: 5}
	resp, err := server.handleGetArticles(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.EqualValues(t, 11, resp.Body.Total)
	assert.Len(t, resp.Body.Articles, 5)
	assert.Equal(t, 2, resp.Body.Page)
	assert.Equal(t, 5, resp.Body.Limit)
}

func TestHandleDeleteArticle_Success(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	// Create an article to delete
	user := &models.User{Email: "test@example.com", Role: models.WRITE}
	article, _, err := db.CreateArticleWithDraft(context.Background(), "To Be Deleted", user.Email)
	require.NoError(t, err)

	admin := &models.User{Email: "admin@test.com", Role: models.ADMIN}
	ctx := contextWithUser(admin)

	input := &ArticleSlugInput{Slug: article.Slug}
	resp, err := server.handleDeleteArticle(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusNoContent, resp.Status)

	deletedArticle, err := db.GetArticleBySlug(context.Background(), article.Slug)
	require.NoError(t, err)
	assert.Nil(t, deletedArticle)
}

func TestHandleDeleteArticle_Unauthorized(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	user := &models.User{Email: "test@example.com", Role: models.WRITE}
	ctx := contextWithUser(user)

	input := &ArticleSlugInput{Slug: "home"}
	resp, err := server.handleDeleteArticle(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	humaErr, ok := err.(*huma.ErrorModel)
	require.True(t, ok)
	assert.Equal(t, 403, humaErr.Status)
}

func TestHandleDeleteArticle_NotFound(t *testing.T) {
	db := newTestDB(t)
	server := newTestServer(t, db)

	admin := &models.User{Email: "admin@test.com", Role: models.ADMIN}
	ctx := contextWithUser(admin)

	input := &ArticleSlugInput{Slug: "non-existent"}
	resp, err := server.handleDeleteArticle(ctx, input)
	require.Error(t, err)
	require.Nil(t, resp)

	humaErr, ok := err.(*huma.ErrorModel)
	require.True(t, ok)
	assert.Equal(t, 404, humaErr.Status)
}
