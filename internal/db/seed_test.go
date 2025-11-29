package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wikilite/pkg/models"
)

func TestIsSeeded_Basic(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	seeded, err := db.IsSeeded(ctx)
	require.NoError(t, err)
	assert.False(t, seeded)
}

func TestSeed_Basic(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	seeded, err := db.IsSeeded(ctx)
	require.NoError(t, err)
	assert.False(t, seeded)

	adminUser := &models.User{
		Name:  "Admin User",
		Email: "admin@example.com",
		Role:  models.ADMIN,
	}
	err = db.Seed(ctx, adminUser, "My Wiki")
	require.NoError(t, err)

	seeded, err = db.IsSeeded(ctx)
	require.NoError(t, err)
	assert.True(t, seeded)

	user, err := db.GetUserByEmail(ctx, "admin@example.com")
	require.NoError(t, err)
	assert.Equal(t, "Admin User", user.Name)
	assert.Equal(t, models.ADMIN, user.Role)
	assert.NotZero(t, user.Id)

	article, err := db.GetArticleBySlug(ctx, "home")
	require.NoError(t, err)
	assert.Equal(t, "My Wiki", article.Title)
	assert.Equal(t, "home", article.Slug)
	assert.Equal(t, 0, article.Version)
	assert.Contains(t, article.Data, "Welcome to your My Wiki")
}
