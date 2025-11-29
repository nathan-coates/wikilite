package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
	"wikilite/pkg/models"

	"github.com/jellydator/ttlcache/v3"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/uptrace/bun"
)

// CreateArticleWithDraft initializes a new Article at Version 0 (Empty)
// and immediately creates the first Draft for it.
func (d *DB) CreateArticleWithDraft(
	ctx context.Context,
	title string,
	userID string,
) (*models.Article, *models.Draft, error) {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}

	defer func(tx bun.Tx) {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Println(err)
		}
	}(tx)

	article := &models.Article{
		Title:     title,
		Version:   0,
		Data:      "",
		CreatedBy: userID,
		CreatedAt: time.Now(),
	}

	_, err = tx.NewInsert().Model(article).Exec(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Pass 'tx' as the executor
	draft, err := d.createGenesisDraft(ctx, tx, article.Id, userID)
	if err != nil {
		return nil, nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, nil, err
	}

	return article, draft, nil
}

// GetArticleBySlug fetches the latest version of an article by its URL slug.
func (d *DB) GetArticleBySlug(ctx context.Context, slug string) (*models.Article, error) {
	item := d.articleCache.Get(slug)
	if item != nil {
		return item.Value(), nil
	}

	article := new(models.Article)
	err := d.NewSelect().
		Model(article).
		Where("slug = ?", slug).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	d.articleCache.Set(slug, article, ttlcache.DefaultTTL)

	return article, nil
}

// GetArticleByID fetches the latest version of an article by ID.
func (d *DB) GetArticleByID(ctx context.Context, id int) (*models.Article, error) {
	article := new(models.Article)

	err := d.NewSelect().
		Model(article).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	if article.Slug != "" {
		d.articleCache.Set(article.Slug, article, ttlcache.DefaultTTL)
	}

	return article, nil
}

// GetArticlesByUser returns a summary list of articles created by a specific user.
func (d *DB) GetArticlesByUser(ctx context.Context, userID string) ([]*models.Article, error) {
	var articles []*models.Article
	err := d.NewSelect().
		Model(&articles).
		Column("id", "title", "slug", "version", "created_by", "created_at").
		Where("created_by = ?", userID).
		Order("created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return articles, nil
}

// GetArticles returns a paginated list of articles.
func (d *DB) GetArticles(ctx context.Context, limit, offset int) ([]*models.Article, int64, error) {
	var articles []*models.Article
	count, err := d.NewSelect().
		Model(&articles).
		Column("id", "title", "slug", "version", "created_by", "created_at").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		ScanAndCount(ctx)

	if err != nil {
		return nil, 0, err
	}

	return articles, int64(count), nil
}

// GetArticleVersion reconstructs a specific version of an article.
func (d *DB) GetArticleVersion(
	ctx context.Context,
	articleID int,
	targetVersion int,
) (string, error) {
	var maxVersion sql.NullInt64
	err := d.NewSelect().
		Model((*models.History)(nil)).
		ColumnExpr("MAX(version)").
		Where("article_id = ?", articleID).
		Where("version > 0").
		Scan(ctx, &maxVersion)

	if err != nil {
		return "", err
	}

	if !maxVersion.Valid || targetVersion > int(maxVersion.Int64) {
		return "", sql.ErrNoRows
	}

	var history []models.History
	err = d.NewSelect().
		Model(&history).
		Where("article_id = ?", articleID).
		Where("version <= ?", targetVersion).
		Order("version ASC").
		Scan(ctx)

	if err != nil {
		return "", err
	}

	dmp := diffmatchpatch.New()
	currentText := ""

	for _, h := range history {
		patches, err := dmp.PatchFromText(h.Data)
		if err != nil {
			return "", fmt.Errorf("failed to parse patch for v%d: %w", h.Version, err)
		}

		currentText, _ = dmp.PatchApply(patches, currentText)
	}

	return currentText, nil
}

// GetArticleHistory returns the versions for an article.
func (d *DB) GetArticleHistory(ctx context.Context, articleID int) ([]*models.History, error) {
	var history []*models.History
	err := d.NewSelect().
		Model(&history).
		Column("id", "article_id", "version", "created_at").
		Where("article_id = ?", articleID).
		Where("version > 0").
		Order("version DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return history, nil
}

// DeleteArticle permanently removes an article and all its associated data.
func (d *DB) DeleteArticle(ctx context.Context, articleID int) error {
	article := new(models.Article)
	err := d.NewSelect().
		Model(article).
		Column("slug").
		Where("id = ?", articleID).
		Scan(ctx)

	targetSlug := ""
	if err == nil {
		targetSlug = article.Slug
	}

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

	_, err = tx.NewDelete().
		Model((*models.Draft)(nil)).
		Where("article_id = ?", articleID).
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = tx.NewDelete().
		Model((*models.History)(nil)).
		Where("article_id = ?", articleID).
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = tx.NewDelete().
		Model((*models.Link)(nil)).
		Where("parent_article_id = ? OR linked_article_id = ?", articleID, articleID).
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = tx.NewDelete().
		Model((*models.Article)(nil)).
		Where("id = ?", articleID).
		Exec(ctx)
	if err != nil {
		return err
	}

	if targetSlug != "" {
		d.articleCache.Delete(targetSlug)
	}

	return tx.Commit()
}
