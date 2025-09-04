package smdclient

import (
	"encoding/json"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type OpenCHAMINodeWithGroups struct {
	cistore.OpenCHAMIComponent
	Groups []string `json:"groups,omitempty" yaml:"groups,omitempty"`
}

func AddNodeToInventoryHandler(f *FakeSMDClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var addNode OpenCHAMINodeWithGroups
		err := json.NewDecoder(r.Body).Decode(&addNode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = f.AddNodeToInventory(addNode.OpenCHAMIComponent)
		if err != nil {
			log.Error().Err(err).Msg("Failed to add node to inventory")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = f.AddNodeToGroups(addNode.ID, addNode.Groups)
		if err != nil {
			log.Error().Err(err).Msg("Failed to add node to groups")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Location", r.URL.Path+"/"+addNode.ID)

	}
}

func ListNodesHandler(f *FakeSMDClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodes := f.ListNodes()
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(nodes)
		if err != nil {
			log.Error().Err(err).Msg("Failed to encode nodes")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func UpdateNodeHandler(f *FakeSMDClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		log.Info().Str("id", id).Msg("Updating node")
		var updateNode OpenCHAMINodeWithGroups
		err := json.NewDecoder(r.Body).Decode(&updateNode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		updateNode.ID = id
		err = f.UpdateNode(updateNode.OpenCHAMIComponent)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update node")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = f.AddNodeToGroups(updateNode.ID, updateNode.Groups)
		if err != nil {
			log.Error().Err(err).Msg("Failed to add node to groups")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
