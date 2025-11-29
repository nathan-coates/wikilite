package models

import (
	"context"
	"fmt"
	"time"
	"wikilite/pkg/utils"

	"github.com/uptrace/bun"
)

// Article represents a single wiki page.
type Article struct {
	bun.BaseModel `bun:"table:articles,alias:a"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	Title     string `bun:"title,notnull"       json:"title"`
	Slug      string `bun:"slug,unique,notnull" json:"slug"`
	Data      string `bun:"data,type:text"      json:"data"`
	CreatedBy string `bun:"created_by"          json:"createdBy"`

	History []*History `bun:"rel:has-many,join:id=article_id" json:"history,omitempty"`
	Drafts  []*Draft   `bun:"rel:has-many,join:id=article_id" json:"drafts,omitempty"`

	Id      int `bun:"id,pk,autoincrement" json:"id"`
	Version int `bun:"version,default:0"   json:"version"`
}

// BeforeAppendModel is a hook that runs before a model is inserted or updated.
func (a *Article) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	if a.Slug == "" && a.Title != "" {
		a.Slug = utils.ToKebabCase(a.Title)
	}

	return nil
}

// AfterInsert is a Bun hook triggered after a successful insert.
func (a *Article) AfterInsert(ctx context.Context, _ *bun.InsertQuery) error {
	logger := LoggerFromContext(ctx)
	if logger != nil {
		_ = logger(
			ctx,
			LevelInfo,
			"DATABASE",
			"Article Created",
			fmt.Sprintf("Title: %s (Slug: %s)", a.Title, a.Slug),
		)
	}
	return nil
}

// AfterUpdate is a Bun hook triggered after a successful update.
func (a *Article) AfterUpdate(ctx context.Context, _ *bun.UpdateQuery) error {
	logger := LoggerFromContext(ctx)
	if logger != nil {
		_ = logger(
			ctx,
			LevelInfo,
			"DATABASE",
			"Article Updated",
			fmt.Sprintf("Slug: %s, Version: %d", a.Slug, a.Version),
		)
	}
	return nil
}

// AfterDelete is a Bun hook triggered after a successful delete.
func (a *Article) AfterDelete(ctx context.Context, _ *bun.DeleteQuery) error {
	logger := LoggerFromContext(ctx)
	if logger != nil {
		_ = logger(
			ctx,
			LevelWarning,
			"DATABASE",
			"Article Deleted",
			fmt.Sprintf("Slug: %s (ID: %d)", a.Slug, a.Id),
		)
	}
	return nil
}

// History represents a single version of an article.
type History struct {
	bun.BaseModel `bun:"table:history,alias:h"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	Article *Article `bun:"rel:belongs-to,join:article_id=id" json:"article,omitempty"`
	Data    string   `bun:"data,type:text"                    json:"data"`

	Id        int `bun:"id,pk,autoincrement" json:"id"`
	ArticleId int `bun:"article_id,notnull"  json:"articleId"`
	Version   int `bun:"version,notnull"     json:"version"`
}

// Link represents a link between two articles.
type Link struct {
	bun.BaseModel `bun:"table:links,alias:l"`

	CreatedAt time.Time `bun:"created_at,default:current_timestamp" json:"createdAt"`

	ParentArticleId int `bun:"parent_article_id,pk" json:"parentId"`
	LinkedArticleId int `bun:"linked_article_id,pk" json:"linkedId"`
}
