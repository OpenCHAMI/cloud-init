package bss

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// BootParamsHandler handles HTTP requests for boot parameters
type BootParamsHandler struct {
	store Store
}

// NewBootParamsHandler creates a new BootParamsHandler
func NewBootParamsHandler(store Store) *BootParamsHandler {
	return &BootParamsHandler{
		store: store,
	}
}

// CreateBootParams handles POST requests to create new boot parameters
func (h *BootParamsHandler) CreateBootParams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var params BootParams
	if err := json.Unmarshal(body, &params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := params.ParseFromJSON(body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get ID from query parameter
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	// Create new boot parameters
	if err := h.store.Set(id, &params); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the stored params to include version in response
	storedParams, err := h.store.Get(id)
	if err != nil {
		http.Error(w, "Failed to retrieve stored parameters", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(storedParams)
}

// UpdateBootParams handles PUT requests to update existing boot parameters
func (h *BootParamsHandler) UpdateBootParams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var params BootParams
	if err := json.Unmarshal(body, &params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := params.ParseFromJSON(body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get ID from query parameter
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	// Update existing boot parameters
	if err := h.store.Update(id, &params); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the updated params to include version in response
	updatedParams, err := h.store.Get(id)
	if err != nil {
		http.Error(w, "Failed to retrieve stored parameters", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedParams)
}

// GetBootParams handles GET requests to retrieve boot parameters
func (h *BootParamsHandler) GetBootParams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	params, err := h.store.Get(id)
	if err != nil {
		http.Error(w, "Boot parameters not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(params)
}

// GenerateBootScriptHandler handles GET requests to generate boot scripts
func (h *BootParamsHandler) GenerateBootScriptHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	params, err := h.store.Get(id)
	if err != nil {
		http.Error(w, "Boot parameters not found", http.StatusNotFound)
		return
	}

	retry := 0
	if retryStr := r.URL.Query().Get("retry"); retryStr != "" {
		if _, err := fmt.Sscanf(retryStr, "%d", &retry); err != nil {
			http.Error(w, "Invalid retry parameter", http.StatusBadRequest)
			return
		}
	}

	arch := r.URL.Query().Get("arch")

	script, err := GenerateBootScript(params, retry, arch)
	if err != nil {
		http.Error(w, "Failed to generate boot script", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(script))
}
