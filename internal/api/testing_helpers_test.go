package api

import (
	"context"
	"os"
	"testing"
	"wikilite/internal/db"
	"wikilite/pkg/models"

	"github.com/stretchr/testify/require"
)

// newTestDB creates a new DB instance with a temporary file-based Database.
func newTestDB(t *testing.T) *db.DB {
	t.Helper()

	mainDBFile := "test_main.db"
	logDBFile := "test_log.db"

	_ = os.Remove(mainDBFile)
	_ = os.Remove(logDBFile)

	database, err := db.New(mainDBFile, logDBFile)
	require.NoError(t, err, "Failed to create new test DB")
	require.NotNil(t, database, "DB object should not be nil")

	adminUser := &models.User{
		Name:  "Admin",
		Email: "admin@test.com",
		Role:  models.ADMIN,
	}
	require.NoError(
		t,
		database.Seed(context.Background(), adminUser, "Home"),
		"Failed to seed Database",
	)

	t.Cleanup(func() {
		_ = database.Close()
		_ = os.Remove(mainDBFile)
		_ = os.Remove(logDBFile)
	})

	return database
}

// newTestServer creates a new server instance for testing.
func newTestServer(t *testing.T, database *db.DB) *Server {
	t.Helper()

	config := ServerConfig{
		Database:          database,
		JwtSecret:         "test-secret",
		JwksURL:           "",
		JwtIssuer:         "",
		JwtEmailClaim:     "",
		WikiName:          "Test Wiki",
		PluginPath:        "",
		PluginStoragePath: "",
		JsPkgsPath:        "",
		Production:        false,
		TrustProxyHeaders: false,
	}

	server, err := NewServer(config)
	require.NoError(t, err, "Failed to create new test server")
	require.NotNil(t, server, "Server object should not be nil")

	return server
}

// newTestServerWithPlugins creates a new server instance with a plugin manager for testing.
// It handles cleanup of the plugin storage Database and plugin manager.
func newTestServerWithPlugins(t *testing.T, database *db.DB, pluginPath string) *Server {
	t.Helper()

	// Use a temporary plugin storage path for testing
	tempPluginStorage := t.TempDir() + "/plugin_storage.db"

	config := ServerConfig{
		Database:          database,
		JwtSecret:         "test-secret",
		JwksURL:           "",
		JwtIssuer:         "",
		JwtEmailClaim:     "",
		WikiName:          "Test Wiki",
		PluginPath:        pluginPath,
		PluginStoragePath: tempPluginStorage,
		JsPkgsPath:        "",
		Production:        false,
		TrustProxyHeaders: false,
	}

	server, err := NewServer(config)
	require.NoError(t, err, "Failed to create new test server with plugins")
	require.NotNil(t, server, "Server object should not be nil")

	t.Cleanup(func() {
		if server.PluginManager != nil {
			_ = server.PluginManager.Close()
		}
		_ = os.Remove(tempPluginStorage)
	})

	return server
}

func contextWithUser(user *models.User) context.Context {
	return context.WithValue(context.Background(), userContextKey, user)
}
