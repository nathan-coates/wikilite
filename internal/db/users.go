package db

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strconv"
	"time"
	"wikilite/pkg/models"

	"github.com/uptrace/bun"
)

// CreateUser registers a new user.
func (d *DB) CreateUser(ctx context.Context, user *models.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	if user.Role == 0 {
		user.Role = models.READ
	}

	_, err := d.NewInsert().Model(user).Exec(ctx)

	return err
}

// GetUserByID fetches a user by their numeric ID.
func (d *DB) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	user := new(models.User)
	err := d.NewSelect().
		Model(user).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return user, nil
}

// GetUserByEmail fetches a user by their email.
func (d *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := new(models.User)
	err := d.NewSelect().
		Model(user).
		Where("email = ?", email).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return user, nil
}

// UpdateUser allows updating specific fields of a user.
func (d *DB) UpdateUser(ctx context.Context, user *models.User, columns ...string) error {
	user.UpdatedAt = time.Now()

	columns = append(columns, "updated_at")

	_, err := d.NewUpdate().
		Model(user).
		Column(columns...).
		WherePK().
		Exec(ctx)

	return err
}

// DeleteUser performs a "Safe Delete".
func (d *DB) DeleteUser(ctx context.Context, id int) error {
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

	userIDStr := strconv.Itoa(id)

	_, err = tx.NewDelete().
		Model((*models.Draft)(nil)).
		Where("created_by = ?", userIDStr).
		Exec(ctx)
	if err != nil {
		return err
	}

	_, err = tx.NewDelete().
		Model((*models.User)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return err
	}

	return tx.Commit()
}
