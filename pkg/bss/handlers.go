package bss

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/OpenCHAMI/cloud-init/pkg/smdclient"
	"github.com/go-chi/chi/v5"
)

// CreateBootParamsHandler creates an HTTP handler for creating boot parameters
func CreateBootParamsHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// Generate a unique ID for the boot parameters
		id := generateBootParamsID()

		// Create new boot parameters
		if err := store.Set(id, &params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Get the stored params to include version in response
		storedParams, err := store.Get(id)
		if err != nil {
			http.Error(w, "Failed to retrieve stored parameters", http.StatusInternalServerError)
			return
		}

		// Set Location header with the new resource URL
		w.Header().Set("Location", fmt.Sprintf("/bootparams/%s", id))
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(storedParams)
	}
}

// UpdateBootParamsHandler creates an HTTP handler for updating boot parameters
func UpdateBootParamsHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get ID from URL parameters
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "ID is required in path", http.StatusBadRequest)
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

		// Update existing boot parameters
		if err := store.Update(id, &params); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Get the updated params to include version in response
		updatedParams, err := store.Get(id)
		if err != nil {
			http.Error(w, "Failed to retrieve stored parameters", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(updatedParams)
	}
}

// GetBootParamsHandler creates an HTTP handler for retrieving boot parameters
func GetBootParamsHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get ID from URL parameters
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "ID is required in path", http.StatusBadRequest)
			return
		}

		params, err := store.Get(id)
		if err != nil {
			http.Error(w, "Boot parameters not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(params)
	}
}

// GenerateBootScriptHandler creates an HTTP handler for generating boot scripts
func GenerateBootScriptHandler(store Store, smd *smdclient.SMDClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var urlId string = chi.URLParam(r, "id")
		var id = urlId
		var err error
		// If this request includes an id, it can be interrpreted as an impersonation request
		if urlId == "" {
			ip := getActualRequestIP(r)
			// Get the component information from the SMD client
			id, err = smd.IDfromIP(ip)
			if err != nil {
				w.WriteHeader(http.StatusUnprocessableEntity)
				return
			} else {
				log.Printf("xname %s with ip %s found\n", id, ip)
			}
		}
		groups, err := smd.GroupMembership(id)
		if err != nil {
			http.Error(w, "Failed to get group membership for "+id, http.StatusInternalServerError)
			return
		}

		var templates []*BootParams
		for _, group := range groups {
			template, err := store.GetTemplateForGroup(group)
			if err != nil {
				http.Error(w, "Failed to get template for group "+group, http.StatusInternalServerError)
				return
			}
			templates = append(templates, template)
		}

		if len(templates) == 0 {
			http.Error(w, "No templates found for any groups", http.StatusNotFound)
			return
		}

		merged, err := MergeBootParams(templates)
		if err != nil {
			http.Error(w, "Failed to merge boot parameters", http.StatusInternalServerError)
			return
		}

		retry := 0
		if retryStr := r.URL.Query().Get("retry"); retryStr != "" {
			if _, err := strconv.Atoi(retryStr); err != nil {
				http.Error(w, "Invalid retry parameter", http.StatusBadRequest)
				return
			}
		}

		arch := r.URL.Query().Get("arch")

		script, err := GenerateIPXEScript(merged, retry, arch)
		if err != nil {
			http.Error(w, "Failed to generate boot script", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(script))
	}
}

// generateBootParamsID generates a Boot Parameters ID in the format "bp-XXXXXX",
// where "XXXXXX" is a random 6-digit hexadecimal string.
func generateBootParamsID() string {
	randBytes := make([]byte, 3)
	rand.Read(randBytes)
	return fmt.Sprintf("bp-%x", randBytes)
}

func getActualRequestIP(r *http.Request) string {
	var ip string
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// If X-Forwarded-For exists, use the first IP in the list
		ip = strings.Split(xff, ",")[0]
	} else {
		// Otherwise, use the remote address
		portIndex := strings.LastIndex(r.RemoteAddr, ":")
		if portIndex > 0 {
			ip = r.RemoteAddr[:portIndex]
		} else {
			ip = r.RemoteAddr
		}
	}
	return strings.TrimSpace(ip)
}
