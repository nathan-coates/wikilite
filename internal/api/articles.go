package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"
	"wikilite/pkg/models"

	"github.com/danielgtaylor/huma/v2"
)

const articleTemplateStr = `
<article id="wiki-content" 
	data-id="{{.Id}}" 
	data-version="{{.Version}}" 
	data-title="{{.Title}}" 
	{{if .Author}}data-author="{{.Author}}"{{end}}>
{{.Content}}
</article>`

// ArticleSlugInput represents the input for getting an article by slug.
type ArticleSlugInput struct {
	Slug string `doc:"The URL slug of the article" path:"slug"`
}

// ArticleContentInput represents the input for getting an article's content.
type ArticleContentInput struct {
	Slug   string `doc:"The URL slug of the article"   path:"slug"`
	Format string `doc:"Output format: 'html' or 'md'"             default:"html" enum:"html,md" query:"format"`
}

// ArticleVersionInput represents the input for getting a specific version of an article.
type ArticleVersionInput struct {
	Slug    string `doc:"The URL slug of the article"             path:"slug"`
	Format  string `doc:"Output format: 'html' or 'md'"                          default:"html" enum:"html,md" query:"format"`
	Version int    `doc:"The specific version number to retrieve" path:"version"`
}

// CreateArticleInput represents the input for creating a new article.
type CreateArticleInput struct {
	Body struct {
		Title string `doc:"Title of the new article" json:"title" required:"true"`
	}
}

// CreateArticleOutput represents the output after creating a new article.
type CreateArticleOutput struct {
	Body struct {
		ArticleSlug string `json:"articleSlug"`
		ArticleId   int    `json:"articleId"`
		DraftID     int    `json:"draftId"`
	}
}

// ArticleListInput represents the input for listing articles by user.
type ArticleListInput struct {
	Email string `doc:"Filter by user email (Admin only). Defaults to current user." query:"email" required:"false"`
}

// ArticlePaginationInput represents the input for paginating articles.
type ArticlePaginationInput struct {
	Page  int `default:"1"  doc:"Page number"    minimum:"1" query:"page"`
	Limit int `default:"10" doc:"Items per page" minimum:"1" query:"limit" maximum:"100"`
}

// PublicArticle is a sanitized version of models.Article for API responses.
type PublicArticle struct {
	CreatedAt time.Time `json:"createdAt"`
	Author    *string   `json:"author,omitempty"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Data      string    `json:"data,omitempty"`
	Id        int       `json:"id"`
	Version   int       `json:"version"`
}

// ArticleListOutput represents the output for a list of articles.
type ArticleListOutput struct {
	Body struct {
		Articles []*PublicArticle `json:"articles"`
	}
}

// ArticleHistoryOutput represents the output for an article's history.
type ArticleHistoryOutput struct {
	Body struct {
		History []*models.History `json:"history"`
	}
}

// ArticleOutput represents the output for a single article.
type ArticleOutput struct {
	Body struct {
		*PublicArticle
	}
}

// PaginatedArticleListOutput represents the output for a paginated list of articles.
type PaginatedArticleListOutput struct {
	Body struct {
		Articles []*PublicArticle `json:"articles"`
		Total    int64            `json:"total"`
		Page     int              `json:"page"`
		Limit    int              `json:"limit"`
	}
}

// registerArticleRoutes registers the article routes with the API.
func (s *Server) registerArticleRoutes() {
	huma.Register(s.api, huma.Operation{
		OperationID: "create-article",
		Method:      http.MethodPost,
		Path:        "/api/articles",
		Summary:     "Create Article",
		Description: "Creates a new article and an initial draft.",
		Tags:        []string{"Articles"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleCreateArticle)

	huma.Register(s.api, huma.Operation{
		OperationID: "get-article",
		Method:      http.MethodGet,
		Path:        "/api/articles/{slug}",
		Summary:     "Get Article (JSON)",
		Tags:        []string{"Articles"},
	}, s.handleGetArticleJSON)

	huma.Register(s.api, huma.Operation{
		OperationID: "get-article-content",
		Method:      http.MethodGet,
		Path:        "/api/articles/{slug}/content",
		Summary:     "Get Article Content",
		Tags:        []string{"Articles"},
	}, s.handleGetArticleContent)

	huma.Register(s.api, huma.Operation{
		OperationID: "get-article-version",
		Method:      http.MethodGet,
		Path:        "/api/articles/{slug}/versions/{version}",
		Summary:     "Get Article Version Content",
		Description: "Retrieve specific version content as HTML or Markdown.",
		Tags:        []string{"Articles"},
	}, s.handleGetArticleVersion)

	huma.Register(s.api, huma.Operation{
		OperationID: "list-orphaned-articles",
		Method:      http.MethodGet,
		Path:        "/api/articles/orphans",
		Summary:     "List Orphaned Articles",
		Description: "Get a list of articles that are not linked to by any other article.",
		Tags:        []string{"Articles"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleGetOrphans)

	huma.Register(s.api, huma.Operation{
		OperationID: "get-user-articles",
		Method:      http.MethodGet,
		Path:        "/api/user/articles",
		Summary:     "Get User Articles",
		Description: "Get articles created by the current user. Admins can filter by email.",
		Tags:        []string{"Articles"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleGetArticlesByUser)

	huma.Register(s.api, huma.Operation{
		OperationID: "get-article-history",
		Method:      http.MethodGet,
		Path:        "/api/articles/{slug}/history",
		Summary:     "Get Article History",
		Tags:        []string{"Articles"},
	}, s.handleGetArticleHistory)

	huma.Register(s.api, huma.Operation{
		OperationID: "list-articles",
		Method:      http.MethodGet,
		Path:        "/api/articles",
		Summary:     "List Articles",
		Description: "Get a paginated list of articles (lightweight, no content).",
		Tags:        []string{"Articles"},
	}, s.handleGetArticles)

	huma.Register(s.api, huma.Operation{
		OperationID: "delete-article",
		Method:      http.MethodDelete,
		Path:        "/api/articles/{slug}",
		Summary:     "Delete Article",
		Tags:        []string{"Articles"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleDeleteArticle)
}

// sanitizeArticle converts a DB model to a safe API response.
func sanitizeArticle(a *models.Article, isAdmin bool) *PublicArticle {
	var author *string

	if isAdmin {
		val := a.CreatedBy
		author = &val
	}

	return &PublicArticle{
		Id:        a.Id,
		Title:     a.Title,
		Slug:      a.Slug,
		Version:   a.Version,
		Data:      a.Data,
		Author:    author,
		CreatedAt: a.CreatedAt,
	}
}

// handleCreateArticle handles the creation of a new article.
func (s *Server) handleCreateArticle(
	ctx context.Context,
	input *CreateArticleInput,
) (*CreateArticleOutput, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	article, draft, err := s.db.CreateArticleWithDraft(ctx, input.Body.Title, user.Email)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to create article", err)
	}

	resp := &CreateArticleOutput{}
	resp.Body.ArticleId = article.Id
	resp.Body.ArticleSlug = article.Slug
	resp.Body.DraftID = draft.Id

	return resp, nil
}

// handleGetArticleJSON handles the request to get an article in JSON format.
func (s *Server) handleGetArticleJSON(
	ctx context.Context,
	input *ArticleSlugInput,
) (*ArticleOutput, error) {
	article, err := s.db.GetArticleBySlug(ctx, input.Slug)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if article == nil {
		return nil, huma.Error404NotFound("Article not found")
	}

	isAdmin := false
	user := getAdminUserFromContext(ctx)
	if user != nil {
		isAdmin = true
	}

	resp := &ArticleOutput{}
	resp.Body.PublicArticle = sanitizeArticle(article, isAdmin)

	return resp, nil
}

// handleGetArticleContent handles the request to get an article's content.
func (s *Server) handleGetArticleContent(
	ctx context.Context,
	input *ArticleContentInput,
) (*huma.StreamResponse, error) {
	article, err := s.db.GetArticleBySlug(ctx, input.Slug)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if article == nil {
		return nil, huma.Error404NotFound("Article not found")
	}

	isAdmin := false
	user := getAdminUserFromContext(ctx)
	if user != nil {
		isAdmin = true
	}

	safeArticle := sanitizeArticle(article, isAdmin)

	if input.Format == "md" {
		return s.streamMarkdown(safeArticle), nil
	}

	return s.streamHTML(
		safeArticle,
	), nil
}

// handleGetOrphans handles the request to get orphaned articles.
func (s *Server) handleGetOrphans(ctx context.Context, _ *struct{}) (*ArticleListOutput, error) {
	user := getAdminUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error403Forbidden("Only admins can view orphaned articles")
	}

	articles, err := s.db.GetOrphanedArticles(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	safeArticles := make([]*PublicArticle, len(articles))
	for i, a := range articles {
		safeArticles[i] = sanitizeArticle(a, true)
	}

	resp := &ArticleListOutput{}
	resp.Body.Articles = safeArticles

	return resp, nil
}

// handleGetArticleVersion handles the request to get a specific version of an article.
func (s *Server) handleGetArticleVersion(
	ctx context.Context,
	input *ArticleVersionInput,
) (*huma.StreamResponse, error) {
	article, err := s.db.GetArticleBySlug(ctx, input.Slug)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if article == nil {
		return nil, huma.Error404NotFound("Article not found")
	}

	content, err := s.db.GetArticleVersion(ctx, article.Id, input.Version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error404NotFound("Article version not found")
		}
		return nil, huma.Error500InternalServerError("Failed to reconstruct version", err)
	}

	versionedArticle := *article
	versionedArticle.Data = content
	versionedArticle.Version = input.Version

	isAdmin := false
	user := getAdminUserFromContext(ctx)
	if user != nil {
		isAdmin = true
	}

	safeArticle := sanitizeArticle(&versionedArticle, isAdmin)

	if input.Format == "md" {
		return s.streamMarkdown(safeArticle), nil
	}

	return s.streamHTML(
		safeArticle,
	), nil
}

// handleGetArticlesByUser handles the request to get articles by user.
func (s *Server) handleGetArticlesByUser(
	ctx context.Context,
	input *ArticleListInput,
) (*ArticleListOutput, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	targetEmail := user.Email

	isAdmin := false
	if user.Role == models.ADMIN {
		isAdmin = true
	}

	if input.Email != "" {
		if !isAdmin {
			return nil, huma.Error403Forbidden("Only admins can view other users' articles")
		}

		targetEmail = input.Email
	}

	articles, err := s.db.GetArticlesByUser(ctx, targetEmail)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	safeArticles := make([]*PublicArticle, len(articles))
	for i, a := range articles {
		safeArticles[i] = sanitizeArticle(a, true)
	}

	resp := &ArticleListOutput{}
	resp.Body.Articles = safeArticles

	return resp, nil
}

// handleGetArticleHistory handles the request to get an article's history.
func (s *Server) handleGetArticleHistory(
	ctx context.Context,
	input *ArticleSlugInput,
) (*ArticleHistoryOutput, error) {
	article, err := s.db.GetArticleBySlug(ctx, input.Slug)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if article == nil {
		return nil, huma.Error404NotFound("Article not found")
	}

	history, err := s.db.GetArticleHistory(ctx, article.Id)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch history", err)
	}

	resp := &ArticleHistoryOutput{}
	resp.Body.History = history

	return resp, nil
}

// handleGetArticles handles the request to get a paginated list of articles.
func (s *Server) handleGetArticles(
	ctx context.Context,
	input *ArticlePaginationInput,
) (*PaginatedArticleListOutput, error) {
	if input.Page < 1 {
		input.Page = 1
	}

	if input.Limit < 1 {
		input.Limit = 10
	}

	offset := (input.Page - 1) * input.Limit

	articles, total, err := s.db.GetArticles(ctx, input.Limit, offset)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	isAdmin := false
	user := getAdminUserFromContext(ctx)
	if user != nil {
		isAdmin = true
	}

	safeArticles := make([]*PublicArticle, len(articles))
	for i, a := range articles {
		safeArticles[i] = sanitizeArticle(a, isAdmin)
	}

	resp := &PaginatedArticleListOutput{}
	resp.Body.Articles = safeArticles
	resp.Body.Total = total
	resp.Body.Page = input.Page
	resp.Body.Limit = input.Limit

	return resp, nil
}

// handleDeleteArticle handles the request to delete an article.
func (s *Server) handleDeleteArticle(
	ctx context.Context,
	input *ArticleSlugInput,
) (*struct{ Status int }, error) {
	admin := getAdminUserFromContext(ctx)
	if admin == nil {
		return nil, huma.Error403Forbidden("Only admins can delete articles")
	}

	article, err := s.db.GetArticleBySlug(ctx, input.Slug)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if article == nil {
		return nil, huma.Error404NotFound("Article not found")
	}

	err = s.db.DeleteArticle(ctx, article.Id)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to delete article", err)
	}

	return &struct{ Status int }{Status: http.StatusNoContent}, nil
}
