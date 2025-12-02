package models

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// BackupCode represents a backup code for OTP authentication.
type BackupCode struct {
	bun.BaseModel `bun:"table:backup_codes,alias:bc"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	Code string `bun:"code,notnull,unique" json:"code"`

	Id     int  `bun:"id,pk,autoincrement" json:"id"`
	UserId int  `bun:"user_id,notnull"     json:"userId"`
	Used   bool `bun:"used,default:false"  json:"used"`
}

// AfterInsert is a Bun hook triggered after a successful insert.
func (b *BackupCode) AfterInsert(ctx context.Context, _ *bun.InsertQuery) error {
	logger := LoggerFromContext(ctx)
	if logger != nil {
		_ = logger(
			ctx,
			LevelInfo,
			"DATABASE",
			"Backup Code Created",
			fmt.Sprintf("Backup Code for User ID: %d", b.UserId),
		)
	}
	return nil
}

// AfterUpdate is a Bun hook triggered after a successful update.
func (b *BackupCode) AfterUpdate(ctx context.Context, _ *bun.UpdateQuery) error {
	logger := LoggerFromContext(ctx)
	if logger != nil {
		_ = logger(
			ctx,
			LevelInfo,
			"DATABASE",
			"Backup Code Updated",
			fmt.Sprintf("Backup Code ID: %d for User ID: %d", b.Id, b.UserId),
		)
	}
	return nil
}

// AfterDelete is a Bun hook triggered after a successful delete.
func (b *BackupCode) AfterDelete(ctx context.Context, _ *bun.DeleteQuery) error {
	logger := LoggerFromContext(ctx)
	if logger != nil {
		_ = logger(
			ctx,
			LevelInfo,
			"DATABASE",
			"Backup Code Deleted",
			fmt.Sprintf("Backup Code ID: %d for User ID: %d", b.Id, b.UserId),
		)
	}
	return nil
}
