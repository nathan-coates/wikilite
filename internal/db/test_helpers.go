package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jellydator/ttlcache/v3"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"wikilite/pkg/models"
)

// newTestDB creates a fresh in-memory database for testing
func newTestDB(t *testing.T) *DB {
	sqldb, err := sql.Open(sqliteshim.ShimName, ":memory:")
	require.NoError(t, err)
	require.NoError(t, sqldb.Ping())

	bunDB := bun.NewDB(sqldb, sqlitedialect.New())

	modelsToCreate := []any{
		(*models.Article)(nil),
		(*models.Draft)(nil),
		(*models.User)(nil),
		(*models.History)(nil),
		(*models.SystemLog)(nil),
		(*models.Link)(nil),
	}

	for _, model := range modelsToCreate {
		_, err = bunDB.NewCreateTable().Model(model).IfNotExists().Exec(context.Background())
		require.NoError(t, err)
	}

	cache := ttlcache.New[string, *models.Article](
		ttlcache.WithTTL[string, *models.Article](cacheTtl),
		ttlcache.WithCapacity[string, *models.Article](cacheSize),
	)
	go cache.Start()

	logChan := make(chan *models.SystemLog, 100)

	bunDB.WithQueryHook(&dbLogger{logChan: logChan})

	db := &DB{
		DB:           bunDB,
		logDB:        bunDB,
		articleCache: cache,
		logChan:      logChan,
	}

	db.startLogWorkers(1)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
