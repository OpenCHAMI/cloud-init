package bss

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// testStoreImplementation is a helper function that runs all store tests against a given Store implementation
func testStoreImplementation(t *testing.T, store Store) {
	// Test initial store state
	params, err := store.Get("test")
	assert.Error(t, err)
	assert.Nil(t, params)

	// Test storing new parameters
	testParams := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
	}

	err = store.Set("test", testParams)
	assert.NoError(t, err)

	// Test retrieving stored parameters
	params, err = store.Get("test")
	assert.NoError(t, err)
	assert.Equal(t, testParams.Kernel, params.Kernel)
	assert.Equal(t, testParams.Initrd, params.Initrd)
	assert.Equal(t, 1, params.Version)

	// Test getting version 1 explicitly
	params, err = store.GetVersion("test", 1)
	assert.NoError(t, err)
	assert.Equal(t, testParams.Kernel, params.Kernel)
	assert.Equal(t, testParams.Initrd, params.Initrd)
	assert.Equal(t, 1, params.Version)

	// Test getting default version (should be 1 initially)
	params, err = store.GetDefault("test")
	assert.NoError(t, err)
	assert.Equal(t, testParams.Kernel, params.Kernel)
	assert.Equal(t, testParams.Initrd, params.Initrd)
	assert.Equal(t, 1, params.Version)

	// Test storing with existing ID
	err = store.Set("test", testParams)
	assert.Error(t, err)

	// Test updating parameters
	updatedParams := &BootParams{
		Kernel: "http://example.com/new-kernel",
		Initrd: "http://example.com/new-initrd",
	}

	err = store.Update("test", updatedParams)
	assert.NoError(t, err)

	// Test retrieving updated parameters (should get version 2)
	params, err = store.Get("test")
	assert.NoError(t, err)
	assert.Equal(t, updatedParams.Kernel, params.Kernel)
	assert.Equal(t, updatedParams.Initrd, params.Initrd)
	assert.Equal(t, 2, params.Version)

	// Test getting version 1 is still available
	params, err = store.GetVersion("test", 1)
	assert.NoError(t, err)
	assert.Equal(t, testParams.Kernel, params.Kernel)
	assert.Equal(t, testParams.Initrd, params.Initrd)
	assert.Equal(t, 1, params.Version)

	// Test getting version 2 explicitly
	params, err = store.GetVersion("test", 2)
	assert.NoError(t, err)
	assert.Equal(t, updatedParams.Kernel, params.Kernel)
	assert.Equal(t, updatedParams.Initrd, params.Initrd)
	assert.Equal(t, 2, params.Version)

	// Test setting default version
	err = store.SetDefault("test", 1)
	assert.NoError(t, err)

	// Test getting default version (should now be 1)
	params, err = store.GetDefault("test")
	assert.NoError(t, err)
	assert.Equal(t, testParams.Kernel, params.Kernel)
	assert.Equal(t, testParams.Initrd, params.Initrd)
	assert.Equal(t, 1, params.Version)

	// Test setting invalid default version
	err = store.SetDefault("test", 3)
	assert.Error(t, err)

	// Test getting invalid version
	params, err = store.GetVersion("test", 3)
	assert.Error(t, err)
	assert.Nil(t, params)

	// Test updating non-existent parameters
	err = store.Update("nonexistent", updatedParams)
	assert.Error(t, err)

	// Test getting non-existent version
	params, err = store.GetVersion("nonexistent", 1)
	assert.Error(t, err)
	assert.Nil(t, params)

	// Test getting default version of non-existent parameters
	params, err = store.GetDefault("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, params)

	// Test setting default version of non-existent parameters
	err = store.SetDefault("nonexistent", 1)
	assert.Error(t, err)
}

// TestMemoryStore runs the store tests against the MemoryStore implementation
func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	testStoreImplementation(t, store)
}

// TestFileStore runs the store tests against the FileStore implementation
// func TestFileStore(t *testing.T) {
//     store := NewFileStore()
//     testStoreImplementation(t, store)
// }
