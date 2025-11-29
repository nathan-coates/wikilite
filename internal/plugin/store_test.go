//go:build plugins

package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBoltStore(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := newBoltStore(dbPath)
	require.NoError(t, err)
	require.NotNil(t, store)

	err = store.Close()
	require.NoError(t, err)

	dbPath2 := filepath.Join(t.TempDir(), "test")
	store2, err := newBoltStore(dbPath2)
	require.NoError(t, err)
	require.NotNil(t, store2)

	err = store2.Close()
	require.NoError(t, err)

	_, err = os.Stat(dbPath2 + ".db")
	assert.NoError(t, err)
}

func TestNewBoltStore_Error(t *testing.T) {
	invalidPath := "/nonexistent/directory/test.db"
	store, err := newBoltStore(invalidPath)
	assert.Error(t, err)
	assert.Nil(t, store)
	assert.Contains(t, err.Error(), "failed to open plugin db")
}

func TestBoltStore_SetAndGet(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	pluginID := "test-plugin"
	key := "test-key"
	value := "test-value"

	err := store.Set(pluginID, key, value)
	require.NoError(t, err)

	retrieved, err := store.Get(pluginID, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestBoltStore_Get_NotFound(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	val, err := store.Get("test-plugin", "non-existent-key")
	require.NoError(t, err)
	assert.Empty(t, val)
}

func TestBoltStore_Get_NonExistentPlugin(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	val, err := store.Get("non-existent-plugin", "any-key")
	require.NoError(t, err)
	assert.Empty(t, val)
}

func TestBoltStore_Set_Overwrite(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	pluginID := "test-plugin"
	key := "test-key"
	value1 := "first-value"
	value2 := "second-value"

	err := store.Set(pluginID, key, value1)
	require.NoError(t, err)

	err = store.Set(pluginID, key, value2)
	require.NoError(t, err)

	retrieved, err := store.Get(pluginID, key)
	require.NoError(t, err)
	assert.Equal(t, value2, retrieved)
}

func TestBoltStore_Delete(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	pluginID := "test-plugin"
	key := "test-key"
	value := "test-value"

	err := store.Set(pluginID, key, value)
	require.NoError(t, err)

	retrieved, err := store.Get(pluginID, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	err = store.Delete(pluginID, key)
	require.NoError(t, err)

	retrieved, err = store.Get(pluginID, key)
	require.NoError(t, err)
	assert.Empty(t, retrieved)
}

func TestBoltStore_Delete_NonExistentKey(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	err := store.Delete("test-plugin", "non-existent-key")
	require.NoError(t, err)
}

func TestBoltStore_Delete_NonExistentPlugin(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	err := store.Delete("non-existent-plugin", "any-key")
	require.NoError(t, err)
}

func TestBoltStore_List(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	pluginID := "test-plugin"
	prefix := "prefix-"

	keys := []string{
		prefix + "key1",
		prefix + "key2",
		prefix + "key3",
		"different-key",
	}

	for i, key := range keys {
		value := "value-" + string(rune('A'+i))
		err := store.Set(pluginID, key, value)
		require.NoError(t, err)
	}

	listed, err := store.List(pluginID, prefix)
	require.NoError(t, err)
	assert.Len(t, listed, 3)

	for _, key := range keys[:3] {
		assert.Contains(t, listed, key)
	}

	assert.NotContains(t, listed, "different-key")
}

func TestBoltStore_List_EmptyPrefix(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	pluginID := "test-plugin"

	keys := []string{"key1", "key2", "key3"}
	for i, key := range keys {
		value := "value-" + string(rune('A'+i))
		err := store.Set(pluginID, key, value)
		require.NoError(t, err)
	}

	listed, err := store.List(pluginID, "")
	require.NoError(t, err)
	assert.Len(t, listed, 3)

	for _, key := range keys {
		assert.Contains(t, listed, key)
	}
}

func TestBoltStore_List_NonExistentPlugin(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	listed, err := store.List("non-existent-plugin", "any-prefix")
	require.NoError(t, err)
	assert.Empty(t, listed)
}

func TestBoltStore_List_NoMatches(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	pluginID := "test-plugin"

	err := store.Set(pluginID, "different-key1", "value1")
	require.NoError(t, err)
	err = store.Set(pluginID, "different-key2", "value2")
	require.NoError(t, err)

	listed, err := store.List(pluginID, "prefix-")
	require.NoError(t, err)
	assert.Empty(t, listed)
}

func TestBoltStore_MultiplePlugins(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	plugin1 := "plugin1"
	plugin2 := "plugin2"

	err := store.Set(plugin1, "key1", "value1-1")
	require.NoError(t, err)
	err = store.Set(plugin1, "key2", "value1-2")
	require.NoError(t, err)

	err = store.Set(plugin2, "key1", "value2-1")
	require.NoError(t, err)
	err = store.Set(plugin2, "key2", "value2-2")
	require.NoError(t, err)

	val, err := store.Get(plugin1, "key1")
	require.NoError(t, err)
	assert.Equal(t, "value1-1", val)

	val, err = store.Get(plugin1, "key2")
	require.NoError(t, err)
	assert.Equal(t, "value1-2", val)

	val, err = store.Get(plugin2, "key1")
	require.NoError(t, err)
	assert.Equal(t, "value2-1", val)

	val, err = store.Get(plugin2, "key2")
	require.NoError(t, err)
	assert.Equal(t, "value2-2", val)

	keys1, err := store.List(plugin1, "")
	require.NoError(t, err)
	assert.Len(t, keys1, 2)

	keys2, err := store.List(plugin2, "")
	require.NoError(t, err)
	assert.Len(t, keys2, 2)
}

func TestBoltStore_Close(t *testing.T) {
	store := newTestStore(t)

	err := store.Set("test-plugin", "key", "value")
	require.NoError(t, err)

	err = store.Close()
	require.NoError(t, err)
}

func TestBoltStore_ConcurrentAccess(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	pluginID := "test-plugin"

	done := make(chan bool, 2)

	go func() {
		err := store.Set(pluginID, "key1", "value1")
		assert.NoError(t, err)
		done <- true
	}()

	go func() {
		err := store.Set(pluginID, "key2", "value2")
		assert.NoError(t, err)
		done <- true
	}()

	<-done
	<-done

	val, err := store.Get(pluginID, "key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", val)

	val, err = store.Get(pluginID, "key2")
	require.NoError(t, err)
	assert.Equal(t, "value2", val)
}

// newTestStore creates a temporary BoltStore for testing
func newTestStore(t *testing.T) *BoltStore {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := newBoltStore(dbPath)
	require.NoError(t, err)
	return store
}
