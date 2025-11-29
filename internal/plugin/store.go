package plugin

import (
	"bytes"
	"fmt"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

// Store defines the interface for a key-value store used by plugins.
type Store interface {
	Get(pluginID string, key string) (string, error)
	Set(pluginID string, key string, value string) error
	Delete(pluginID string, key string) error
	List(pluginID string, prefix string) ([]string, error)
	Close() error
}

// BoltStore is a file-backed implementation using BoltDB.
type BoltStore struct {
	db *bbolt.DB
}

// newBoltStore opens (or creates) the database file.
func newBoltStore(path string) (*BoltStore, error) {
	if filepath.Ext(path) == "" {
		path = path + ".db"
	}

	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin db: %w", err)
	}

	return &BoltStore{db: db}, nil
}

// Get retrieves a value from the store.
func (s *BoltStore) Get(pluginID string, key string) (string, error) {
	var val string

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(pluginID))
		if b == nil {
			return nil
		}

		v := b.Get([]byte(key))
		if v != nil {
			val = string(v)
		}

		return nil
	})

	return val, err
}

// Set stores a value in the store.
func (s *BoltStore) Set(pluginID string, key string, value string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(pluginID))
		if err != nil {
			return err
		}

		return b.Put([]byte(key), []byte(value))
	})
}

// Delete removes a value from the store.
func (s *BoltStore) Delete(pluginID string, key string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(pluginID))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(key))
	})
}

// List returns all keys starting with the given prefix.
func (s *BoltStore) List(pluginID string, prefix string) ([]string, error) {
	var keys []string

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(pluginID))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		prefixBytes := []byte(prefix)

		for k, _ := c.Seek(prefixBytes); k != nil && bytes.HasPrefix(k, prefixBytes); k, _ = c.Next() {
			keys = append(keys, string(k))
		}

		return nil
	})

	return keys, err
}

// Close closes the database.
func (s *BoltStore) Close() error {
	return s.db.Close()
}
