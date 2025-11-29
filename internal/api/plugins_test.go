//go:build plugins

package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wikilite/pkg/models"
)

func TestHandlePluginAction_InvalidJSON(t *testing.T) {
	testDB := newTestDB(t)

	server := newTestServer(t, testDB)

	ctx := context.Background()

	input := &PluginActionInput{
		PluginID: "test-plugin",
		Action:   "test-action",
		Body:     map[string]any{"invalid": make(chan int)},
	}

	result, err := server.handlePluginAction(ctx, input)

	require.Error(t, err)
	assert.Nil(t, result)

	var humaErr *huma.ErrorModel
	if errors.As(err, &humaErr) {
		assert.Equal(t, http.StatusBadRequest, humaErr.Status)
	}
}

func TestHandlePluginAction_PluginExecutionError(t *testing.T) {
	testDB := newTestDB(t)

	tempPluginDir := t.TempDir()

	pluginContent := `
// Test plugin that always fails
function onAction(action, payload, ctx) {
	throw new Error("Test plugin error");
}
`
	err := os.WriteFile(filepath.Join(tempPluginDir, "01-test.js"), []byte(pluginContent), 0644)
	require.NoError(t, err)

	server := newTestServerWithPlugins(t, testDB, tempPluginDir)

	user := &models.User{Id: 1, Email: "test@example.com"}
	ctx := contextWithUser(user)

	input := &PluginActionInput{
		PluginID: "nonexistent-plugin",
		Action:   "test-action",
		Body:     map[string]any{"message": "hello"},
	}

	result, err := server.handlePluginAction(ctx, input)

	require.Error(t, err)
	assert.Nil(t, result)

	var humaErr *huma.ErrorModel
	if errors.As(err, &humaErr) {
		assert.Equal(t, http.StatusInternalServerError, humaErr.Status)
		assert.Contains(t, humaErr.Detail, "Plugin execution error")
	}

	time.Sleep(100 * time.Millisecond)

	logs, _, err := testDB.GetLogs(ctx, 10, 0, models.LevelError)
	require.NoError(t, err)
	assert.Greater(t, len(logs), 0)
	found := false
	for _, log := range logs {
		if log.Source == "plugin-action" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should have a plugin-action log entry")
}

func TestHandlePluginAction_InvalidResponseJSON(t *testing.T) {
	invalidJSON := `{"invalid": json}`

	var result any
	err := json.Unmarshal([]byte(invalidJSON), &result)

	require.Error(t, err)

	fallbackResult := map[string]string{"raw": invalidJSON}
	assert.Equal(t, invalidJSON, fallbackResult["raw"])
}

func TestRegisterPluginRoutes_NilPluginManager(t *testing.T) {
	testDB := newTestDB(t)

	server := newTestServer(t, testDB)

	tempPluginDir := t.TempDir()
	tempStoragePath := t.TempDir() + "/plugin_storage.db"

	err := server.registerPluginRoutes(tempPluginDir, tempStoragePath, "")
	require.NoError(t, err)

	require.NotNil(t, server.PluginManager)
}

func TestRegisterPluginRoutes_NoPlugins(t *testing.T) {
	testDB := newTestDB(t)

	server := newTestServer(t, testDB)

	tempPluginDir := t.TempDir()
	tempStoragePath := t.TempDir() + "/plugin_storage.db"

	err := server.registerPluginRoutes(tempPluginDir, tempStoragePath, "")
	require.NoError(t, err)

	require.NotNil(t, server.PluginManager)
	require.Empty(t, server.PluginManager.Plugins)
}

func TestPluginActionInput_Validation(t *testing.T) {
	input := &PluginActionInput{
		PluginID: "test-plugin",
		Action:   "test-action",
		Body:     map[string]any{"message": "hello"},
		Slug:     "test-article",
	}

	jsonBytes, err := json.Marshal(input)
	require.NoError(t, err)

	var unmarshaled PluginActionInput
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, input.PluginID, unmarshaled.PluginID)
	assert.Equal(t, input.Action, unmarshaled.Action)
	assert.Equal(t, input.Slug, unmarshaled.Slug)
	assert.Equal(t, "hello", unmarshaled.Body["message"])
}

func TestPluginActionOutput_Validation(t *testing.T) {
	output := &PluginActionOutput{
		Body: map[string]any{"result": "success", "data": 123},
	}

	jsonBytes, err := json.Marshal(output)
	require.NoError(t, err)

	var unmarshaled PluginActionOutput
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	bodyMap, ok := unmarshaled.Body.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "success", bodyMap["result"])
	assert.Equal(t, float64(123), bodyMap["data"]) // JSON numbers become float64
}

func TestExecutePlugins_LoggerFunction(t *testing.T) {
	ctx := context.Background()

	var loggedMessages []string
	mockLogger := func(ctx context.Context, level models.LogLevel, source string, message string, data string) error {
		loggedMessages = append(loggedMessages, message)
		return nil
	}

	pluginErrs := []error{
		errors.New("first error"),
		errors.New("second error"),
	}

	for _, err := range pluginErrs {
		_ = mockLogger(ctx, models.LevelError, "plugin", "Error executing plugin: "+err.Error(), "")
	}

	assert.Len(t, loggedMessages, 2)
	assert.Contains(t, loggedMessages[0], "first error")
	assert.Contains(t, loggedMessages[1], "second error")
}
