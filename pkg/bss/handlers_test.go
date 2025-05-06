package bss

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenCHAMI/cloud-init/pkg/smdclient"
)

func TestCreateBootParamsHandler(t *testing.T) {
	store := NewMemoryStore()
	handler := CreateBootParamsHandler(store)

	// Test successful creation
	params := BootParams{
		Kernel: "vmlinuz",
		Initrd: "initrd.img",
		Params: "console=ttyS0",
	}
	body, _ := json.Marshal(params)
	req := httptest.NewRequest("POST", "/bootparams", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	// Verify Location header
	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Expected Location header to be set")
	}

	// Test invalid JSON
	req = httptest.NewRequest("POST", "/bootparams", bytes.NewBuffer([]byte("invalid json")))
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetBootParamsHandler(t *testing.T) {
	store := NewMemoryStore()
	handler := GetBootParamsHandler(store)

	// Create test data
	params := BootParams{
		Kernel: "vmlinuz",
		Initrd: "initrd.img",
		Params: "console=ttyS0",
	}
	id := "test-id"
	store.Set(id, &params)

	// Test successful retrieval
	req := httptest.NewRequest("GET", "/bootparams/"+id, nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test missing ID
	req = httptest.NewRequest("GET", "/bootparams/", nil)
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Test non-existent ID
	req = httptest.NewRequest("GET", "/bootparams/nonexistent", nil)
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUpdateBootParamsHandler(t *testing.T) {
	store := NewMemoryStore()
	handler := UpdateBootParamsHandler(store)

	// Create test data
	params := BootParams{
		Kernel: "vmlinuz",
		Initrd: "initrd.img",
		Params: "console=ttyS0",
	}
	id := "test-id"
	store.Set(id, &params)

	// Test successful update
	updatedParams := BootParams{
		Kernel: "vmlinuz-new",
		Initrd: "initrd.img-new",
		Params: "console=ttyS0,debug",
	}
	body, _ := json.Marshal(updatedParams)
	req := httptest.NewRequest("PUT", "/bootparams/"+id, bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test missing ID
	req = httptest.NewRequest("PUT", "/bootparams/", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Test non-existent ID
	req = httptest.NewRequest("PUT", "/bootparams/nonexistent", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest("PUT", "/bootparams/"+id, bytes.NewBuffer([]byte("invalid json")))
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGenerateBootScriptHandler(t *testing.T) {
	store := NewMemoryStore()
	client, err := smdclient.NewSMDClient("http://localhost:8080", "test", "test", "test", "test", false)
	if err != nil {
		t.Fatalf("Failed to create SMD client: %v", err)
	}
	handler := GenerateBootScriptHandler(store, client)

	// Create test data
	params := BootParams{
		Kernel: "vmlinuz",
		Initrd: "initrd.img",
		Params: "console=ttyS0",
	}
	id := "test-id"
	store.Set(id, &params)

	// Test successful script generation
	req := httptest.NewRequest("GET", "/bootscript/"+id, nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test missing ID
	req = httptest.NewRequest("GET", "/bootscript/", nil)
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Test non-existent ID
	req = httptest.NewRequest("GET", "/bootscript/nonexistent", nil)
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	// Test with retry parameter
	req = httptest.NewRequest("GET", "/bootscript/"+id+"?retry=3", nil)
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test with invalid retry parameter
	req = httptest.NewRequest("GET", "/bootscript/"+id+"?retry=invalid", nil)
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
