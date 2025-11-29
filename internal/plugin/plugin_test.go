package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugins")
	require.NoError(t, err)
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpDir)

	plugin1 := "10-plugin-one.js"
	plugin2 := "2-plugin-two.js"
	invalidPlugin := "invalid-plugin.js"
	plugin1Content := "console.log('one');"
	plugin2Content := "console.log('two');"

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, plugin1), []byte(plugin1Content), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, plugin2), []byte(plugin2Content), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, invalidPlugin), []byte("..."), 0644))

	plugins, err := loadFromDirectory(tmpDir)
	require.NoError(t, err)

	require.Len(t, plugins, 2)

	assert.Equal(t, "plugin-two", plugins[0].ID)
	assert.Equal(t, 2, plugins[0].Order)
	assert.Equal(t, plugin2Content, plugins[0].Script)

	assert.Equal(t, "plugin-one", plugins[1].ID)
	assert.Equal(t, 10, plugins[1].Order)
	assert.Equal(t, plugin1Content, plugins[1].Script)
}

func TestEnsureTypeDefinitions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugins_types")
	require.NoError(t, err)
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tmpDir)

	typesPath := filepath.Join(tmpDir, "types.d.ts")

	err = ensureTypeDefinitions(tmpDir)
	require.NoError(t, err)

	_, err = os.Stat(typesPath)
	assert.NoError(t, err, "types.d.ts should have been created")

	content, err := os.ReadFile(typesPath)
	require.NoError(t, err)
	assert.Equal(t, typeDefinitionContent, string(content))

	err = ensureTypeDefinitions(tmpDir)
	require.NoError(t, err, "Calling ensureTypeDefinitions again should not produce an error")
}
