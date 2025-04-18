package bss

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()

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

	// Test retrieving updated parameters
	params, err = store.Get("test")
	assert.NoError(t, err)
	assert.Equal(t, updatedParams.Kernel, params.Kernel)
	assert.Equal(t, updatedParams.Initrd, params.Initrd)
	assert.Equal(t, 2, params.Version)

	// Test updating non-existent parameters
	err = store.Update("nonexistent", updatedParams)
	assert.Error(t, err)
}
