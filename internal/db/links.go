package db

import (
	"context"
	"wikilite/pkg/models"
	"wikilite/pkg/utils"

	"github.com/uptrace/bun"
)

// updateArticleLinks updates the links for an article based on the content.
func (d *DB) updateArticleLinks(
	ctx context.Context,
	tx bun.IDB,
	parentArticleID int,
	content string,
) error {
	foundSlugs := utils.ExtractSlugsFromContent(content)

	if len(foundSlugs) == 0 {
		_, err := tx.NewDelete().
			Model((*models.Link)(nil)).
			Where("parent_article_id = ?", parentArticleID).
			Exec(ctx)

		return err
	}

	var targetArticles []models.Article

	err := tx.NewSelect().
		Model(&targetArticles).
		Column("id").
		Where("slug IN (?)", bun.In(foundSlugs)).
		Scan(ctx)
	if err != nil {
		return err
	}

	newLinks := make([]*models.Link, 0, len(targetArticles))

	for _, target := range targetArticles {
		if target.Id == parentArticleID {
			continue
		}

		newLinks = append(newLinks, &models.Link{
			ParentArticleId: parentArticleID,
			LinkedArticleId: target.Id,
		})
	}

	_, err = tx.NewDelete().
		Model((*models.Link)(nil)).
		Where("parent_article_id = ?", parentArticleID).
		Exec(ctx)
	if err != nil {
		return err
	}

	if len(newLinks) > 0 {
		_, err = tx.NewInsert().Model(&newLinks).Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetOrphanedArticles returns articles that are NOT linked to by any other article.
func (d *DB) GetOrphanedArticles(ctx context.Context) ([]*models.Article, error) {
	var orphans []*models.Article

	subquery := d.NewSelect().
		Model((*models.Link)(nil)).
		Column("linked_article_id").
		Distinct()

	err := d.NewSelect().
		Model(&orphans).
		Where("id NOT IN (?)", subquery).
		Where("slug != 'home'").
		Order("title ASC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return orphans, nil
}
