package memstore

import (
	"testing"

	storetesting "github.com/OpenCHAMI/cloud-init/pkg/cistore/testing"
)

func TestMemStore(t *testing.T) {
	// Create a new MemStore instance
	store := NewMemStore()

	// Create a cleanup function that will be called in all cases
	cleanup := func() {
		// No cleanup needed for MemStore as it's in-memory
	}

	// Run the standard test suite
	storetesting.RunStoreTests(t, store, cleanup)
}
