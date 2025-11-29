package db

import (
	"context"
	"time"
	"wikilite/pkg/models"
)

// CreateLogEntry pushes a log entry to the worker pool.
func (d *DB) CreateLogEntry(
	ctx context.Context,
	level models.LogLevel,
	source, message, data string,
) error {
	logEntry := &models.SystemLog{
		Level:     level,
		Source:    source,
		Message:   message,
		Data:      data,
		Duration:  0,
		CreatedAt: time.Now(),
	}

	select {
	case d.logChan <- logEntry:
		return nil
	default:
		return nil
	}
}

// GetLogs fetches logs with optional filtering.
func (d *DB) GetLogs(
	ctx context.Context,
	limit int,
	offset int,
	level models.LogLevel,
) ([]*models.SystemLog, int64, error) {
	var logs []*models.SystemLog
	query := d.logDB.NewSelect().
		Model(&logs).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset)

	if level != "" {
		query.Where("level = ?", level)
	}

	count, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return logs, int64(count), nil
}

// GetLogByID fetches a single log entry details.
func (d *DB) GetLogByID(ctx context.Context, id int64) (*models.SystemLog, error) {
	logEntry := new(models.SystemLog)

	err := d.logDB.NewSelect().Model(logEntry).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return logEntry, nil
}

// PruneLogs removes old log entries from the database.
func (d *DB) PruneLogs(ctx context.Context, age time.Duration) (int64, error) {
	cutoff := time.Now().Add(-age)

	res, err := d.logDB.NewDelete().
		Model((*models.SystemLog)(nil)).
		Where("created_at < ?", cutoff).
		Exec(ctx)

	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}
