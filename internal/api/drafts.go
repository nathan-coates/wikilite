package api

import (
	"context"
	"errors"
	"net/http"
	"time"
	"wikilite/internal/db"
	"wikilite/pkg/models"

	"github.com/danielgtaylor/huma/v2"
)

// DraftIDInput represents the input for getting a draft by ID.
type DraftIDInput struct {
	ID int `doc:"The ID of the draft" path:"id"`
}

// ArticleSlugForDraftInput represents the input for creating a draft for an article.
type ArticleSlugForDraftInput struct {
	Slug string `doc:"The URL slug of the article to create a draft for" path:"slug"`
}

// UpdateDraftInput represents the input for updating a draft.
type UpdateDraftInput struct {
	Body struct {
		Content string `doc:"The full markdown content of the draft" json:"content" required:"true"`
	}
	ID int `doc:"The ID of the draft" path:"id"`
}

// PublicDraft represents a draft with the reconstructed content (not patch).
type PublicDraft struct {
	UpdatedAt      time.Time `json:"updatedAt"`
	ArticleTitle   string    `json:"articleTitle"`
	ArticleSlug    string    `json:"articleSlug"`
	Content        string    `json:"content"`
	Id             int       `json:"id"`
	ArticleId      int       `json:"articleId"`
	ArticleVersion int       `json:"articleVersion"`
}

// DraftOutput represents the output for a single draft.
type DraftOutput struct {
	Body struct {
		Draft *PublicDraft `json:"draft"`
	}
}

// DraftListOutput represents the output for a list of drafts.
type DraftListOutput struct {
	Body struct {
		Drafts []*PublicDraft `json:"drafts"`
	}
}

// registerDraftRoutes registers the draft routes with the API.
func (s *Server) registerDraftRoutes() {
	huma.Register(s.api, huma.Operation{
		OperationID: "create-draft",
		Method:      http.MethodPost,
		Path:        "/api/articles/{slug}/draft",
		Summary:     "Create Draft",
		Description: "Create a new draft for an article. If one exists for this user, it is replaced.",
		Tags:        []string{"Drafts"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleCreateDraft)

	huma.Register(s.api, huma.Operation{
		OperationID: "get-my-drafts",
		Method:      http.MethodGet,
		Path:        "/api/drafts",
		Summary:     "List My Drafts",
		Tags:        []string{"Drafts"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleGetMyDrafts)

	huma.Register(s.api, huma.Operation{
		OperationID: "get-article-drafts",
		Method:      http.MethodGet,
		Path:        "/api/articles/{slug}/drafts",
		Summary:     "List Article Drafts",
		Description: "List all drafts for a specific article. Admins see all; Users see only their own.",
		Tags:        []string{"Drafts"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleGetArticleDrafts)

	huma.Register(s.api, huma.Operation{
		OperationID: "get-draft",
		Method:      http.MethodGet,
		Path:        "/api/drafts/{id}",
		Summary:     "Get Draft",
		Tags:        []string{"Drafts"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleGetDraft)

	huma.Register(s.api, huma.Operation{
		OperationID: "update-draft",
		Method:      http.MethodPut,
		Path:        "/api/drafts/{id}",
		Summary:     "Update Draft",
		Tags:        []string{"Drafts"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleUpdateDraft)

	huma.Register(s.api, huma.Operation{
		OperationID: "publish-draft",
		Method:      http.MethodPost,
		Path:        "/api/drafts/{id}/publish",
		Summary:     "Publish Draft",
		Tags:        []string{"Drafts"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handlePublishDraft)

	huma.Register(s.api, huma.Operation{
		OperationID: "discard-draft",
		Method:      http.MethodDelete,
		Path:        "/api/drafts/{id}",
		Summary:     "Discard Draft",
		Tags:        []string{"Drafts"},
		Security:    []map[string][]string{{"bearer": {}}},
	}, s.handleDiscardDraft)
}

// handleCreateDraft handles the creation of a new draft.
func (s *Server) handleCreateDraft(
	ctx context.Context,
	input *ArticleSlugForDraftInput,
) (*DraftOutput, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	if user.Role < models.WRITE {
		return nil, huma.Error403Forbidden("You do not have permission to edit articles")
	}

	article, err := s.db.GetArticleBySlug(ctx, input.Slug)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if article == nil {
		return nil, huma.Error404NotFound("Article not found")
	}

	draft, err := s.db.CreateDraft(ctx, article.Id, article.Data, user.Email)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to create draft", err)
	}

	resp := &DraftOutput{}
	resp.Body.Draft = &PublicDraft{
		Id:             draft.Id,
		ArticleId:      article.Id,
		ArticleTitle:   article.Title,
		ArticleSlug:    article.Slug,
		ArticleVersion: article.Version,
		Content:        article.Data,
		UpdatedAt:      draft.UpdatedAt,
	}

	return resp, nil
}

// handleGetMyDrafts handles the request to get the current user's drafts.
func (s *Server) handleGetMyDrafts(ctx context.Context, _ *struct{}) (*DraftListOutput, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	drafts, err := s.db.GetDraftsByUser(ctx, user.Email)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	publicDrafts := make([]*PublicDraft, len(drafts))
	for i, d := range drafts {
		publicDrafts[i] = &PublicDraft{
			Id:             d.Id,
			ArticleId:      d.ArticleId,
			ArticleTitle:   d.Article.Title,
			ArticleSlug:    d.Article.Slug,
			ArticleVersion: d.ArticleVersion,
			Content:        "",
			UpdatedAt:      d.UpdatedAt,
		}
	}

	resp := &DraftListOutput{}
	resp.Body.Drafts = publicDrafts

	return resp, nil
}

// handleGetDraft handles the request to get a single draft by ID.
func (s *Server) handleGetDraft(ctx context.Context, input *DraftIDInput) (*DraftOutput, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	draft, content, err := s.db.GetDraftByID(ctx, input.ID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if draft == nil {
		return nil, huma.Error404NotFound("Draft not found")
	}

	if draft.CreatedBy != user.Email && user.Role != models.ADMIN {
		return nil, huma.Error403Forbidden("You can only view your own drafts")
	}

	resp := &DraftOutput{}
	resp.Body.Draft = &PublicDraft{
		Id:             draft.Id,
		ArticleId:      draft.ArticleId,
		ArticleTitle:   draft.Article.Title,
		ArticleSlug:    draft.Article.Slug,
		ArticleVersion: draft.ArticleVersion,
		Content:        content,
		UpdatedAt:      draft.UpdatedAt,
	}

	return resp, nil
}

// handleGetArticleDrafts handles the request to get all drafts for a specific article.
func (s *Server) handleGetArticleDrafts(
	ctx context.Context,
	input *ArticleSlugForDraftInput,
) (*DraftListOutput, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	article, err := s.db.GetArticleBySlug(ctx, input.Slug)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if article == nil {
		return nil, huma.Error404NotFound("Article not found")
	}

	var filterUserID []string
	if user.Role != models.ADMIN {
		filterUserID = []string{user.Email}
	}

	drafts, err := s.db.GetDraftsByArticle(ctx, article.Id, filterUserID...)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	publicDrafts := make([]*PublicDraft, len(drafts))
	for i, d := range drafts {
		publicDrafts[i] = &PublicDraft{
			Id:             d.Id,
			ArticleId:      d.ArticleId,
			ArticleTitle:   article.Title,
			ArticleSlug:    article.Slug,
			ArticleVersion: d.ArticleVersion,
			Content:        "",
			UpdatedAt:      d.UpdatedAt,
		}
	}

	resp := &DraftListOutput{}
	resp.Body.Drafts = publicDrafts

	return resp, nil
}

// handleUpdateDraft handles the request to update a draft.
func (s *Server) handleUpdateDraft(
	ctx context.Context,
	input *UpdateDraftInput,
) (*struct{ Status int }, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	err := s.db.UpdateDraft(ctx, input.ID, input.Body.Content, user.Email)
	if err != nil {
		if errors.Is(err, db.ErrCannotEditDraft) {
			return nil, huma.Error403Forbidden("You can only edit your own drafts")
		}
		return nil, huma.Error500InternalServerError("Failed to update draft", err)
	}

	return &struct{ Status int }{Status: http.StatusNoContent}, nil
}

// handlePublishDraft handles the request to publish a draft.
func (s *Server) handlePublishDraft(
	ctx context.Context,
	input *DraftIDInput,
) (*struct{ Status int }, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	if user.Role < models.WRITE {
		return nil, huma.Error403Forbidden("You do not have permission to publish")
	}

	draft, _, err := s.db.GetDraftByID(ctx, input.ID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error", err)
	}

	if draft.CreatedBy != user.Email && user.Role != models.ADMIN {
		return nil, huma.Error403Forbidden("You cannot publish another user's draft")
	}

	err = s.db.PublishDraft(ctx, input.ID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to publish draft", err)
	}

	return &struct{ Status int }{Status: http.StatusNoContent}, nil
}

// handleDiscardDraft handles the request to discard a draft.
func (s *Server) handleDiscardDraft(
	ctx context.Context,
	input *DraftIDInput,
) (*struct{ Status int }, error) {
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	err := s.db.DiscardDraft(ctx, input.ID, user.Email)
	if err != nil {
		if errors.Is(err, db.ErrCannotDiscardDraft) {
			return nil, huma.Error403Forbidden("You can only discard your own drafts")
		}
		return nil, huma.Error500InternalServerError("Failed to discard draft", err)
	}

	return &struct{ Status int }{Status: http.StatusNoContent}, nil
}
