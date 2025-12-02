package db

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"
	"wikilite/pkg/models"

	"github.com/uptrace/bun"
)

// CreateBackupCode creates a new backup code for a user.
func (d *DB) CreateBackupCode(ctx context.Context, backupCode *models.BackupCode) error {
	backupCode.CreatedAt = time.Now()
	backupCode.UpdatedAt = time.Now()

	_, err := d.NewInsert().Model(backupCode).Exec(ctx)
	return err
}

// CreateBackupCodes creates multiple backup codes for a user in a transaction.
func (d *DB) CreateBackupCodes(ctx context.Context, backupCodes []*models.BackupCode) error {
	if len(backupCodes) == 0 {
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

	for _, backupCode := range backupCodes {
		backupCode.CreatedAt = time.Now()
		backupCode.UpdatedAt = time.Now()
		_, err = tx.NewInsert().Model(backupCode).Exec(ctx)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetBackupCodeByCode fetches a backup code by its code string.
func (d *DB) GetBackupCodeByCode(ctx context.Context, code string) (*models.BackupCode, error) {
	backupCode := new(models.BackupCode)
	err := d.NewSelect().
		Model(backupCode).
		Where("code = ?", code).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return backupCode, nil
}

// GetBackupCodesByUserId fetches all backup codes for a user.
func (d *DB) GetBackupCodesByUserId(ctx context.Context, userId int) ([]*models.BackupCode, error) {
	var backupCodes []*models.BackupCode
	err := d.NewSelect().
		Model(&backupCodes).
		Where("user_id = ?", userId).
		Order("created_at ASC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return backupCodes, nil
}

// GetUnusedBackupCodesByUserId fetches all unused backup codes for a user.
func (d *DB) GetUnusedBackupCodesByUserId(
	ctx context.Context,
	userId int,
) ([]*models.BackupCode, error) {
	var backupCodes []*models.BackupCode
	err := d.NewSelect().
		Model(&backupCodes).
		Where("user_id = ? AND used = ?", userId, false).
		Order("created_at ASC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return backupCodes, nil
}

// UseBackupCode marks a backup code as used.
func (d *DB) UseBackupCode(ctx context.Context, backupCode *models.BackupCode) error {
	backupCode.Used = true
	backupCode.UpdatedAt = time.Now()

	_, err := d.NewUpdate().
		Model(backupCode).
		Column("used", "updated_at").
		WherePK().
		Exec(ctx)

	return err
}

// DeleteBackupCodesByUserId deletes all backup codes for a user.
func (d *DB) DeleteBackupCodesByUserId(ctx context.Context, userId int) error {
	_, err := d.NewDelete().
		Model((*models.BackupCode)(nil)).
		Where("user_id = ?", userId).
		Exec(ctx)

	return err
}

// CountUnusedBackupCodesByUserId counts the number of unused backup codes for a user.
func (d *DB) CountUnusedBackupCodesByUserId(ctx context.Context, userId int) (int, error) {
	count, err := d.NewSelect().
		Model((*models.BackupCode)(nil)).
		Where("user_id = ? AND used = ?", userId, false).
		Count(ctx)

	return count, err
}
