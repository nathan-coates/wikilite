package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wikilite/pkg/models"
)

func TestUpdateArticleLinks_Basic(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	article1, _, err := db.CreateArticleWithDraft(ctx, "Article One", "test@example.com")
	require.NoError(t, err)

	article2, _, err := db.CreateArticleWithDraft(ctx, "Article Two", "test@example.com")
	require.NoError(t, err)

	content := "# Test Article\n\nThis links to [Article Two](/wiki/article-two)."

	err = db.updateArticleLinks(ctx, db.DB, article1.Id, content)
	require.NoError(t, err)

	var links []models.Link
	err = db.NewSelect().Model(&links).Where("parent_article_id = ?", article1.Id).Scan(ctx)
	require.NoError(t, err)
	assert.Len(t, links, 1)
	assert.Equal(t, article2.Id, links[0].LinkedArticleId)
}

func TestGetOrphanedArticles_Basic(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	article1, _, err := db.CreateArticleWithDraft(ctx, "Article One", "test@example.com")
	require.NoError(t, err)

	article2, _, err := db.CreateArticleWithDraft(ctx, "Article Two", "test@example.com")
	require.NoError(t, err)

	article3, _, err := db.CreateArticleWithDraft(ctx, "Article Three", "test@example.com")
	require.NoError(t, err)

	content := "# Article One\n\nLinks to [Article Two](/wiki/article-two)."
	err = db.updateArticleLinks(ctx, db.DB, article1.Id, content)
	require.NoError(t, err)

	orphans, err := db.GetOrphanedArticles(ctx)
	require.NoError(t, err)

	orphanedIds := make(map[int]bool)
	for _, orphan := range orphans {
		orphanedIds[orphan.Id] = true
	}

	assert.False(t, orphanedIds[article2.Id], "Article 2 is linked, should not be orphan")
	assert.True(t, orphanedIds[article3.Id], "Article 3 has no links, should be orphan")
	assert.True(t, orphanedIds[article1.Id], "Article 1 has no incoming links, should be orphan")
}
