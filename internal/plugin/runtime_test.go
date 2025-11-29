//go:build plugins

package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager_Success(t *testing.T) {
	pluginDir, err := os.MkdirTemp("", "plugins")
	require.NoError(t, err)
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(pluginDir)

	dbPath, err := os.MkdirTemp("", "plugindb")
	require.NoError(t, err)
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(dbPath)

	pluginFile := "10-test-plugin.js"
	pluginContent := "console.log('hello from plugin');"
	require.NoError(
		t,
		os.WriteFile(filepath.Join(pluginDir, pluginFile), []byte(pluginContent), 0644),
	)

	manager, err := NewManager(dbPath, pluginDir, "")
	require.NoError(t, err)
	require.NotNil(t, manager)

	assert.True(t, manager.HasPlugins())
	assert.Len(t, manager.Plugins, 1)
	assert.Equal(t, "test-plugin", manager.Plugins[0].ID)

	err = manager.Close()
	assert.NoError(t, err)
}

func TestExecutePipeline_Success(t *testing.T) {
	pluginDir, err := os.MkdirTemp("", "plugins")
	require.NoError(t, err)
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(pluginDir)

	dbPath, err := os.MkdirTemp("", "plugindb")
	require.NoError(t, err)
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(dbPath)

	pluginFile := "10-modifier.js"
	pluginContent := `
		function onArticleRender(content, ctx) {
			return content + " [modified]";
		}
	`
	require.NoError(
		t,
		os.WriteFile(filepath.Join(pluginDir, pluginFile), []byte(pluginContent), 0644),
	)

	manager, err := NewManager(dbPath, pluginDir, "")
	require.NoError(t, err)
	defer func(manager *Manager) {
		_ = manager.Close()
	}(manager)

	initialContent := "Hello, World!"
	modifiedContent, errors, err := manager.ExecutePipeline("onArticleRender", initialContent, nil)
	require.NoError(t, err)
	assert.Empty(t, errors)
	assert.Equal(t, "Hello, World! [modified]", modifiedContent)
}

func TestExecutePluginAction_Success(t *testing.T) {
	pluginDir, err := os.MkdirTemp("", "plugins")
	require.NoError(t, err)
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(pluginDir)

	dbPath, err := os.MkdirTemp("", "plugindb")
	require.NoError(t, err)
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(dbPath)

	pluginFile := "10-action-plugin.js"
	pluginContent := `
		function onAction(action, payload, ctx) {
			if (action === "my-action") {
				return { result: "action-success" };
			}
		}
	`
	require.NoError(
		t,
		os.WriteFile(filepath.Join(pluginDir, pluginFile), []byte(pluginContent), 0644),
	)

	manager, err := NewManager(dbPath, pluginDir, "")
	require.NoError(t, err)
	defer func(manager *Manager) {
		_ = manager.Close()
	}(manager)

	result, err := manager.ExecutePluginAction(
		"action-plugin",
		"my-action",
		`{"data": "test"}`,
		nil,
	)
	require.NoError(t, err)
	assert.Contains(t, result, "action-success")
}

func TestExecutePipeline_WithCache(t *testing.T) {
	pluginDir, err := os.MkdirTemp("", "plugins")
	require.NoError(t, err)
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(pluginDir)

	dbPath, err := os.MkdirTemp("", "plugindb")
	require.NoError(t, err)
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(dbPath)

	pluginFile := "10-caching-plugin.js"
	pluginContent := `
		function onArticleRender(content, ctx) {
			return content + " " + Math.random();
		}
	`
	require.NoError(
		t,
		os.WriteFile(filepath.Join(pluginDir, pluginFile), []byte(pluginContent), 0644),
	)

	manager, err := NewManager(dbPath, pluginDir, "")
	require.NoError(t, err)
	defer func(manager *Manager) {
		_ = manager.Close()
	}(manager)

	initialContent := "cached content"
	contextData := map[string]any{"Slug": "test-slug"}

	result1, _, err := manager.ExecutePipeline("onArticleRender", initialContent, contextData)
	require.NoError(t, err)

	result2, _, err := manager.ExecutePipeline("onArticleRender", initialContent, contextData)
	require.NoError(t, err)

	assert.Equal(t, result1, result2)

	result3, _, err := manager.ExecutePipeline("onArticleRender", "different content", contextData)
	require.NoError(t, err)
	assert.NotEqual(t, result1, result3)
}
