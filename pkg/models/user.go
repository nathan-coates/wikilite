package models

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// UserRole defines the permission level of a user.
type UserRole int

const (
	// READ grants permission to view articles.
	READ UserRole = iota + 1
	// WRITE grants permission to create and edit articles.
	WRITE
	// ADMIN grants all permissions, including user management.
	ADMIN
)

// User represents a user account.
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	Name      string    `bun:"name,notnull"                                          json:"name"`
	Email     string    `bun:"email,unique,notnull"                                  json:"email"`
	Hash      string    `bun:"hash"                                                  json:"-"`
	OTPSecret string    `bun:"otp_secret"                                            json:"-"`

	Id         int      `bun:"id,pk,autoincrement"               json:"id"`
	Role       UserRole `bun:"role,notnull"                      json:"role"`
	IsExternal bool     `bun:"is_external,notnull,default:false" json:"isExternal"`
	Disabled   bool     `bun:"disabled,default:false"            json:"disabled"`
}

// AfterInsert is a Bun hook triggered after a successful insert.
func (u *User) AfterInsert(ctx context.Context, _ *bun.InsertQuery) error {
	logger := LoggerFromContext(ctx)
	if logger != nil {
		_ = logger(
			ctx,
			LevelInfo,
			"DATABASE",
			"User Created",
			fmt.Sprintf("User: %s (ID: %d, Role: %d)", u.Email, u.Id, u.Role),
		)
	}
	return nil
}

// AfterUpdate is a Bun hook triggered after a successful update.
func (u *User) AfterUpdate(ctx context.Context, _ *bun.UpdateQuery) error {
	logger := LoggerFromContext(ctx)
	if logger != nil {
		_ = logger(
			ctx,
			LevelInfo,
			"DATABASE",
			"User Updated",
			fmt.Sprintf("User: %s (ID: %d)", u.Email, u.Id),
		)
	}
	return nil
}

// AfterDelete is a Bun hook triggered after a successful delete.
func (u *User) AfterDelete(ctx context.Context, _ *bun.DeleteQuery) error {
	logger := LoggerFromContext(ctx)
	if logger != nil {
		_ = logger(
			ctx,
			LevelWarning,
			"DATABASE",
			"User Deleted",
			fmt.Sprintf("User: %s (ID: %d)", u.Email, u.Id),
		)
	}
	return nil
}
