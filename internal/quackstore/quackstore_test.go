package quackstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	storetesting "github.com/OpenCHAMI/cloud-init/pkg/cistore/testing"
	"github.com/stretchr/testify/assert"
)

func TestQuackStore(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "quackstore-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the database file path
	dbPath := filepath.Join(tmpDir, "test.db")

	// Test database creation and schema initialization
	t.Run("Database Initialization", func(t *testing.T) {
		store, err := NewQuackStore(dbPath)
		assert.NoError(t, err)
		defer store.Close()

		// Verify database file was created
		_, err = os.Stat(dbPath)
		assert.NoError(t, err)

		// Verify tables were created by trying to insert test data
		_, err = store.db.Exec("INSERT INTO groups (name, data) VALUES (?, ?)", "init-test", "{}")
		assert.NoError(t, err)

		_, err = store.db.Exec("INSERT INTO instances (node_name, data) VALUES (?, ?)", "init-test", "{}")
		assert.NoError(t, err)

		_, err = store.db.Exec("INSERT INTO cluster_defaults (id, data) VALUES (?, ?)", 1, "{}")
		assert.NoError(t, err)
	})

	// Test database cleanup
	t.Run("Database Cleanup", func(t *testing.T) {
		// Remove the existing database file to start fresh
		os.Remove(dbPath)

		store, err := NewQuackStore(dbPath)
		assert.NoError(t, err)
		defer store.Close()

		// Insert some test data
		_, err = store.db.Exec("INSERT INTO groups (name, data) VALUES (?, ?)", "cleanup-test", "{}")
		assert.NoError(t, err)

		// Close the database
		err = store.Close()
		assert.NoError(t, err)

		// Reopen the database and verify data persists
		store, err = NewQuackStore(dbPath)
		assert.NoError(t, err)
		defer store.Close()

		var count int
		err = store.db.QueryRow("SELECT COUNT(*) FROM groups").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	// Remove the database file before running the standard test suite
	os.Remove(dbPath)

	// Create a new QuackStore instance for the standard test suite
	store, err := NewQuackStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create QuackStore: %v", err)
	}
	defer store.Close()

	// Create a cleanup function that will be called in all cases
	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}
	defer cleanup()

	// Run the standard test suite with a timeout
	done := make(chan bool)
	go func() {
		storetesting.RunStoreTests(t, store, cleanup)
		done <- true
	}()

	select {
	case <-done:
		// Tests completed successfully
	case <-time.After(30 * time.Second):
		t.Fatal("Tests timed out after 30 seconds")
	}
}
