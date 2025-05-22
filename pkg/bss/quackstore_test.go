package bss

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuackStore(t *testing.T) {
	// Create a temporary database file
	dbPath := "test_quackstore.db"
	defer os.Remove(dbPath)

	// Create a new QuackStore instance
	store, err := NewQuackStore(dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Run the standard store tests
	testStoreImplementation(t, store)
}

func TestQuackStoreV1Operations(t *testing.T) {
	// Create a temporary database file
	dbPath := "test_quackstore_v1.db"
	defer os.Remove(dbPath)

	// Create a new QuackStore instance
	store, err := NewQuackStore(dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Test V1 boot parameters
	v1Params := &BootParamsV1{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
		Params: "console=ttyS0",
	}

	// Test setting V1 parameters
	err = store.SetV1("x1000c0s0b0n0", v1Params)
	assert.NoError(t, err)

	// Test getting V1 parameters
	retrieved, err := store.GetV1("x1000c0s0b0n0")
	assert.NoError(t, err)
	assert.Equal(t, v1Params.Kernel, retrieved.Kernel)
	assert.Equal(t, v1Params.Initrd, retrieved.Initrd)
	assert.Equal(t, v1Params.Params, retrieved.Params)

	// Test getting non-existent V1 parameters
	_, err = store.GetV1("nonexistent")
	assert.Error(t, err)
}

func TestQuackStoreGroupTemplateOperations(t *testing.T) {
	// Create a temporary database file
	dbPath := "test_quackstore_templates.db"
	defer os.Remove(dbPath)

	// Create a new QuackStore instance
	store, err := NewQuackStore(dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Create a boot parameter set
	params := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
		Params: "console=ttyS0",
	}

	// Store the boot parameters
	err = store.Set("test-template", params)
	assert.NoError(t, err)

	// Test assigning template to group
	err = store.AssignTemplateToGroup("test-template", "test-group", 1)
	assert.NoError(t, err)

	// Test getting template for group
	template, err := store.GetTemplateForGroup("test-group")
	assert.NoError(t, err)
	assert.Equal(t, params.Kernel, template.Kernel)
	assert.Equal(t, params.Initrd, template.Initrd)
	assert.Equal(t, params.Params, template.Params)

	// Test getting template for non-existent group
	_, err = store.GetTemplateForGroup("nonexistent")
	assert.Error(t, err)

	// Test assigning template to group with non-existent boot parameters
	err = store.AssignTemplateToGroup("nonexistent", "test-group", 1)
	assert.Error(t, err)

	// Test assigning template to group with default version
	err = store.AssignTemplateToGroup("test-template", "test-group-default", 0)
	assert.NoError(t, err)

	// Test getting template for group with default version
	template, err = store.GetTemplateForGroup("test-group-default")
	assert.NoError(t, err)
	assert.Equal(t, params.Kernel, template.Kernel)
	assert.Equal(t, params.Initrd, template.Initrd)
	assert.Equal(t, params.Params, template.Params)
}

func TestQuackStoreVersioning(t *testing.T) {
	// Create a temporary database file
	dbPath := "test_quackstore_versioning.db"
	defer os.Remove(dbPath)

	// Create a new QuackStore instance
	store, err := NewQuackStore(dbPath)
	assert.NoError(t, err)
	defer store.Close()

	// Create initial boot parameters
	initialParams := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
		Params: "console=ttyS0",
	}

	// Store the initial boot parameters
	err = store.Set("test-versioning", initialParams)
	assert.NoError(t, err)

	// Update the boot parameters
	updatedParams := &BootParams{
		Kernel: "http://example.com/new-kernel",
		Initrd: "http://example.com/new-initrd",
		Params: "console=ttyS0,debug",
	}

	err = store.Update("test-versioning", updatedParams)
	assert.NoError(t, err)

	// Test getting the latest version
	latest, err := store.Get("test-versioning")
	assert.NoError(t, err)
	assert.Equal(t, updatedParams.Kernel, latest.Kernel)
	assert.Equal(t, updatedParams.Initrd, latest.Initrd)
	assert.Equal(t, updatedParams.Params, latest.Params)
	assert.Equal(t, 2, latest.Version)

	// Test getting version 1
	v1, err := store.GetVersion("test-versioning", 1)
	assert.NoError(t, err)
	assert.Equal(t, initialParams.Kernel, v1.Kernel)
	assert.Equal(t, initialParams.Initrd, v1.Initrd)
	assert.Equal(t, initialParams.Params, v1.Params)
	assert.Equal(t, 1, v1.Version)

	// Test getting version 2
	v2, err := store.GetVersion("test-versioning", 2)
	assert.NoError(t, err)
	assert.Equal(t, updatedParams.Kernel, v2.Kernel)
	assert.Equal(t, updatedParams.Initrd, v2.Initrd)
	assert.Equal(t, updatedParams.Params, v2.Params)
	assert.Equal(t, 2, v2.Version)

	// Test setting default version
	err = store.SetDefault("test-versioning", 1)
	assert.NoError(t, err)

	// Test getting default version
	defaultParams, err := store.GetDefault("test-versioning")
	assert.NoError(t, err)
	assert.Equal(t, initialParams.Kernel, defaultParams.Kernel)
	assert.Equal(t, initialParams.Initrd, defaultParams.Initrd)
	assert.Equal(t, initialParams.Params, defaultParams.Params)
	assert.Equal(t, 1, defaultParams.Version)

	// Test setting invalid default version
	err = store.SetDefault("test-versioning", 3)
	assert.Error(t, err)

	// Test getting invalid version
	_, err = store.GetVersion("test-versioning", 3)
	assert.Error(t, err)
}
