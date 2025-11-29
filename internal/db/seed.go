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

// IsSeeded checks if the database has already been populated.
func (d *DB) IsSeeded(ctx context.Context) (bool, error) {
	return d.NewSelect().Model((*models.User)(nil)).Exists(ctx)
}

// Seed initializes the database with a default Admin user and a Home page.
func (d *DB) Seed(ctx context.Context, adminUser *models.User, homeTitle string) error {
	seeded, err := d.IsSeeded(ctx)
	if err != nil {
		return fmt.Errorf("failed to check seed status: %w", err)
	}

	if seeded {
		return nil
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

	adminUser.CreatedAt = time.Now()
	adminUser.UpdatedAt = time.Now()

	if adminUser.Role == 0 {
		adminUser.Role = models.ADMIN
	}

	_, err = tx.NewInsert().Model(adminUser).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}

	initialContent := fmt.Sprintf(
		"# Welcome to your %s\n\nThis is the home page of your new wiki.",
		homeTitle,
	)

	adminIDStr := fmt.Sprintf("%d", adminUser.Id)

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain("", initialContent, false)
	patches := dmp.PatchMake("", diffs)
	patchText := dmp.PatchToText(patches)

	article := &models.Article{
		Title:     homeTitle,
		Slug:      "home",
		Version:   0,
		Data:      initialContent,
		CreatedBy: adminIDStr,
		CreatedAt: time.Now(),
	}

	_, err = tx.NewInsert().Model(article).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to seed home article: %w", err)
	}

	history := &models.History{
		ArticleId: article.Id,
		Version:   0,
		Data:      patchText,
		CreatedAt: time.Now(),
	}

	_, err = tx.NewInsert().Model(history).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to seed home history: %w", err)
	}

	return tx.Commit()
}
