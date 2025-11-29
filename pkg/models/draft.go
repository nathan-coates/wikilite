package models

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Draft represents a work-in-progress version of an article.
type Draft struct {
	bun.BaseModel `bun:"table:drafts,alias:d"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	Article   *Article `bun:"rel:belongs-to,join:article_id=id" json:"article,omitempty"`
	Data      string   `bun:"data,type:text"                    json:"data"`
	CreatedBy string   `bun:"created_by"                        json:"createdBy"`

	Id             int `bun:"id,pk,autoincrement" json:"id"`
	ArticleId      int `bun:"article_id,notnull"  json:"articleId"`
	ArticleVersion int `bun:"article_version"     json:"articleVersion"`
}

// AfterInsert is a Bun hook triggered after a successful insert.
func (d *Draft) AfterInsert(ctx context.Context, _ *bun.InsertQuery) error {
	logger := LoggerFromContext(ctx)
	if logger != nil {
		_ = logger(
			ctx,
			LevelInfo,
			"DATABASE",
			"Draft Created",
			fmt.Sprintf("Draft ID: %d for Article ID: %d", d.Id, d.ArticleId),
		)
	}
	return nil
}

// AfterUpdate is a Bun hook triggered after a successful update.
func (d *Draft) AfterUpdate(ctx context.Context, _ *bun.UpdateQuery) error {
	logger := LoggerFromContext(ctx)
	if logger != nil {
		_ = logger(ctx, LevelInfo, "DATABASE", "Draft Updated", fmt.Sprintf("Draft ID: %d", d.Id))
	}
	return nil
}

// AfterDelete is a Bun hook triggered after a successful delete.
func (d *Draft) AfterDelete(ctx context.Context, _ *bun.DeleteQuery) error {
	logger := LoggerFromContext(ctx)
	if logger != nil {
		_ = logger(
			ctx,
			LevelInfo,
			"DATABASE",
			"Draft Discarded/Deleted",
			fmt.Sprintf("Draft ID: %d", d.Id),
		)
	}
	return nil
}
