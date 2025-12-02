package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
	"wikilite/pkg/models"

	"github.com/jellydator/ttlcache/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

const (
	cacheTtl       = 30 * time.Minute
	cacheSize      = 1000
	logChannelSize = 1000
	logWorkers     = 5
	DefaultWikiDb  = "wiki.db"
	DefaultLogDb   = "logs.db"
)

// DB wraps the Bun DB instance and holds the application cache.
type DB struct {
	*bun.DB

	logDB *bun.DB

	articleCache *ttlcache.Cache[string, *models.Article]

	logChan chan *models.SystemLog
	logWg   sync.WaitGroup
}

// dbLogger intercepts main DB queries and sends them to the log channel.
type dbLogger struct {
	logChan chan *models.SystemLog
}

// BeforeQuery is a no-op that satisfies the bun.QueryHook interface.
func (h *dbLogger) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}

// AfterQuery logs the query to the log channel.
func (h *dbLogger) AfterQuery(_ context.Context, event *bun.QueryEvent) {
	query := event.Query
	if len(query) > 1000 {
		query = query[:1000] + "...(truncated)"
	}

	level := models.LevelSQL
	if event.Err != nil {
		level = models.LevelSQLError
	}

	logEntry := &models.SystemLog{
		Level:     level,
		Source:    "DATABASE",
		Message:   fmt.Sprintf("Query execution (%s)", event.Operation()),
		Data:      query,
		Duration:  time.Since(event.StartTime).Milliseconds(),
		CreatedAt: time.Now(),
	}

	select {
	case h.logChan <- logEntry:
	default:
	}
}

// New initializes connections, cache, and the log worker pool.
func New(mainDSN string, logDSN string) (*DB, error) {
	sqldb, err := sql.Open(sqliteshim.ShimName, mainDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open main db: %w", err)
	}

	mainDB := bun.NewDB(sqldb, sqlitedialect.New())

	logSqlDb, err := sql.Open(sqliteshim.ShimName, logDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open log db: %w", err)
	}

	logDB := bun.NewDB(logSqlDb, sqlitedialect.New())

	logChan := make(chan *models.SystemLog, logChannelSize)

	mainDB.WithQueryHook(&dbLogger{logChan: logChan})

	cache := ttlcache.New[string, *models.Article](
		ttlcache.WithTTL[string, *models.Article](cacheTtl),
		ttlcache.WithCapacity[string, *models.Article](cacheSize),
	)
	go cache.Start()

	d := &DB{
		DB:           mainDB,
		logDB:        logDB,
		articleCache: cache,
		logChan:      logChan,
	}

	d.startLogWorkers(logWorkers)

	err = d.createTables(context.Background())
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Ping checks the connectivity of both the main and log databases.
func (d *DB) Ping(_ context.Context) error {
	err := d.DB.Ping()
	if err != nil {
		return fmt.Errorf("main db ping failed: %w", err)
	}

	err = d.logDB.Ping()
	if err != nil {
		return fmt.Errorf("log db ping failed: %w", err)
	}

	return nil
}

// Close cleans up resources.
func (d *DB) Close() error {
	d.articleCache.Stop()

	close(d.logChan)
	d.logWg.Wait()
	_ = d.logDB.Close()

	return d.DB.Close()
}

// createTables creates the necessary database tables if they don't exist.
func (d *DB) createTables(ctx context.Context) error {
	mainModels := []any{
		(*models.Article)(nil),
		(*models.History)(nil),
		(*models.Link)(nil),
		(*models.Draft)(nil),
		(*models.User)(nil),
		(*models.BackupCode)(nil),
	}

	for _, model := range mainModels {
		_, err := d.NewCreateTable().Model(model).IfNotExists().Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create main table: %w", err)
		}
	}

	logModels := []any{
		(*models.SystemLog)(nil),
	}

	for _, model := range logModels {
		_, err := d.logDB.NewCreateTable().Model(model).IfNotExists().Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create log table: %w", err)
		}
	}

	return nil
}

// startLogWorkers spins up 'count' background goroutines to process logs.
func (d *DB) startLogWorkers(count int) {
	for range count {

		d.logWg.Go(func() {

			for entry := range d.logChan {
				_, _ = d.logDB.NewInsert().Model(entry).Exec(context.Background())
			}
		})
	}
}
