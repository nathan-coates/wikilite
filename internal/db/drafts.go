package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
	"wikilite/pkg/models"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/uptrace/bun"
)

// Draft authorization errors
var (
	ErrCannotEditDraft    = errors.New("unauthorized: you cannot edit this draft")
	ErrCannotDiscardDraft = errors.New("unauthorized: you cannot discard this draft")
)

// createGenesisDraft is internal but attached to DB to allow for future logging/metrics.
func (d *DB) createGenesisDraft(
	ctx context.Context,
	db bun.IDB,
	articleID int,
	userID string,
) (*models.Draft, error) {
	draft := &models.Draft{
		ArticleId:      articleID,
		ArticleVersion: 0,
		Data:           "",
		CreatedBy:      userID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := db.NewInsert().Model(draft).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return draft, nil
}

// CreateDraft creates a new Draft.
func (d *DB) CreateDraft(
	ctx context.Context,
	articleID int,
	newContent string,
	userID string,
) (*models.Draft, error) {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer func(tx bun.Tx) {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Println(err)
		}
	}(tx)

	_, err = tx.NewDelete().
		Model((*models.Draft)(nil)).
		Where("article_id = ? AND created_by = ?", articleID, userID).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	article := new(models.Article)

	err = tx.NewSelect().Model(article).Where("id = ?", articleID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(article.Data, newContent, false)
	dmp.DiffCleanupSemantic(diffs)
	patches := dmp.PatchMake(article.Data, diffs)
	patchText := dmp.PatchToText(patches)

	draft := &models.Draft{
		ArticleId:      article.Id,
		ArticleVersion: article.Version,
		Data:           patchText,
		CreatedBy:      userID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err = tx.NewInsert().Model(draft).Exec(ctx)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return draft, nil
}

// GetDraftByID fetches a draft.
func (d *DB) GetDraftByID(ctx context.Context, draftID int) (*models.Draft, string, error) {
	draft := new(models.Draft)
	err := d.NewSelect().
		Model(draft).
		Relation("Article").
		Where("d.id = ?", draftID).
		Scan(ctx)

	if err != nil {
		return nil, "", err
	}

	dmp := diffmatchpatch.New()

	patches, err := dmp.PatchFromText(draft.Data)
	if err != nil {
		return draft, "", fmt.Errorf("failed to parse draft patch: %w", err)
	}

	reconstructedText, results := dmp.PatchApply(patches, draft.Article.Data)

	for _, success := range results {
		if !success {
			return draft, "", fmt.Errorf("version mismatch caused patch conflict")
		}
	}

	return draft, reconstructedText, nil
}

// GetDraftsByUser returns all drafts started by a specific user.
func (d *DB) GetDraftsByUser(ctx context.Context, userID string) ([]*models.Draft, error) {
	var drafts []*models.Draft
	err := d.NewSelect().
		Model(&drafts).
		Relation("Article").
		Where("d.created_by = ?", userID).
		Order("d.updated_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return drafts, nil
}

// GetDraftsByArticle returns all active drafts for a specific article.
func (d *DB) GetDraftsByArticle(
	ctx context.Context,
	articleID int,
	userID ...string,
) ([]*models.Draft, error) {
	var drafts []*models.Draft
	query := d.NewSelect().
		Model(&drafts).
		Where("d.article_id = ?", articleID).
		Order("d.updated_at DESC")

	if len(userID) > 0 {
		query.Where("d.created_by = ?", userID[0])
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	return drafts, nil
}

// UpdateDraft updates the draft with new content.
func (d *DB) UpdateDraft(ctx context.Context, draftID int, newContent string, userID string) error {
	draft := new(models.Draft)

	err := d.NewSelect().Model(draft).Where("id = ?", draftID).Scan(ctx)
	if err != nil {
		return err
	}

	if draft.CreatedBy != userID {
		return ErrCannotEditDraft
	}

	article := new(models.Article)

	err = d.NewSelect().Model(article).Where("id = ?", draft.ArticleId).Scan(ctx)
	if err != nil {
		return err
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(article.Data, newContent, false)
	dmp.DiffCleanupSemantic(diffs)
	patches := dmp.PatchMake(article.Data, diffs)

	if len(patches) == 0 {
		_, err := d.NewDelete().Model(draft).WherePK().Exec(ctx)

		return err
	}

	patchString := dmp.PatchToText(patches)

	draft.Data = patchString
	draft.UpdatedAt = time.Now()
	draft.ArticleVersion = article.Version

	_, err = d.NewUpdate().
		Model(draft).
		Column("data", "updated_at", "article_version").
		WherePK().
		Exec(ctx)

	return err
}

// PublishDraft applies the draft patch to the article.
func (d *DB) PublishDraft(ctx context.Context, draftID int) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func(tx bun.Tx) {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Println(err)
		}
	}(tx)

	draft := new(models.Draft)

	err = tx.NewSelect().Model(draft).Where("id = ?", draftID).Scan(ctx)
	if err != nil {
		return err
	}

	article := new(models.Article)

	err = tx.NewSelect().Model(article).Where("id = ?", draft.ArticleId).Scan(ctx)
	if err != nil {
		return err
	}

	dmp := diffmatchpatch.New()

	patches, err := dmp.PatchFromText(draft.Data)
	if err != nil {
		return fmt.Errorf("invalid patch data: %w", err)
	}

	newText, results := dmp.PatchApply(patches, article.Data)

	for _, success := range results {
		if !success {
			return errors.New("patch failed to apply cleanly")
		}
	}

	history := &models.History{
		ArticleId: article.Id,
		Version:   article.Version + 1,
		Data:      draft.Data,
		CreatedAt: draft.UpdatedAt,
	}

	_, err = tx.NewInsert().Model(history).Exec(ctx)
	if err != nil {
		return err
	}

	article.Data = newText
	article.Version++

	_, err = tx.NewUpdate().Model(article).Column("data", "version").WherePK().Exec(ctx)
	if err != nil {
		return err
	}

	err = d.updateArticleLinks(ctx, tx, article.Id, newText)
	if err != nil {
		return fmt.Errorf("failed to update article links: %w", err)
	}

	d.articleCache.Delete(article.Slug)

	_, err = tx.NewDelete().Model(draft).WherePK().Exec(ctx)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DiscardDraft deletes a draft.
func (d *DB) DiscardDraft(ctx context.Context, draftID int, userID string) error {
	draft := new(models.Draft)
	err := d.NewSelect().
		Model(draft).
		Column("created_by").
		Where("id = ?", draftID).
		Scan(ctx)

	if err != nil {
		return err
	}

	if draft.CreatedBy != userID {
		return ErrCannotDiscardDraft
	}

	_, err = d.NewDelete().Model(draft).Where("id = ?", draftID).Exec(ctx)

	return err
}
