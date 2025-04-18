package bss

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBootParamsHandler(t *testing.T) {
	store := NewMemoryStore()
	handler := NewBootParamsHandler(store)

	// Test data
	testParams := &BootParams{
		Kernel: "http://example.com/kernel",
		Initrd: "http://example.com/initrd",
	}

	// Test creating new parameters
	body, _ := json.Marshal(testParams)
	req := httptest.NewRequest("POST", "/bootparams?id=test", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.CreateBootParams(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var response BootParams
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, testParams.Kernel, response.Kernel)
	assert.Equal(t, testParams.Initrd, response.Initrd)
	assert.Equal(t, 1, response.Version)

	// Test retrieving parameters
	req = httptest.NewRequest("GET", "/bootparams?id=test", nil)
	w = httptest.NewRecorder()
	handler.GetBootParams(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, testParams.Kernel, response.Kernel)
	assert.Equal(t, testParams.Initrd, response.Initrd)
	assert.Equal(t, 1, response.Version)

	// Test updating parameters
	updatedParams := &BootParams{
		Kernel: "http://example.com/new-kernel",
		Initrd: "http://example.com/new-initrd",
	}
	body, _ = json.Marshal(updatedParams)
	req = httptest.NewRequest("PUT", "/bootparams?id=test", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.UpdateBootParams(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, updatedParams.Kernel, response.Kernel)
	assert.Equal(t, updatedParams.Initrd, response.Initrd)
	assert.Equal(t, 2, response.Version)

	// Test generating boot script
	req = httptest.NewRequest("GET", "/bootscript?id=test", nil)
	w = httptest.NewRecorder()
	handler.GenerateBootScriptHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), updatedParams.Kernel)
	assert.Contains(t, w.Body.String(), updatedParams.Initrd)

	// Test error cases
	// Missing ID
	req = httptest.NewRequest("POST", "/bootparams", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.CreateBootParams(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Invalid JSON
	req = httptest.NewRequest("POST", "/bootparams?id=test", bytes.NewReader([]byte("invalid")))
	w = httptest.NewRecorder()
	handler.CreateBootParams(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Non-existent parameters
	req = httptest.NewRequest("GET", "/bootparams?id=nonexistent", nil)
	w = httptest.NewRecorder()
	handler.GetBootParams(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Update non-existent parameters
	req = httptest.NewRequest("PUT", "/bootparams?id=nonexistent", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.UpdateBootParams(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Create with existing ID
	req = httptest.NewRequest("POST", "/bootparams?id=test", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.CreateBootParams(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
