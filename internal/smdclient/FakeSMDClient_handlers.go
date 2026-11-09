// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smdclient

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/openchami/cloud-init/pkg/cistore"
)

type OpenCHAMINodeWithGroups struct {
	cistore.OpenCHAMIComponent
	Groups []string `json:"groups,omitempty" yaml:"groups,omitempty"`
}

// AddNodeToInventoryHandler godoc
//
//	@Summary		Add a fake SMD node
//	@Description	Available only when fake SMD is enabled.
//	@Tags			admin,fake-smd
//	@Accept			json
//	@Param			node	body		OpenCHAMINodeWithGroups	true	"Node data and groups"
//	@Failure		400		{string}	string
//	@Success		201		{object}	nil
//	@Failure		500		{string}	string
//	@Security		BearerAuth
//	@Failure		401	{string}	string	"Missing or invalid bearer JWT"
//	@Failure		503	{string}	string	"Admin JWT verifier unavailable"
//	@Router			/admin/fake-sm/nodes [post]
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

// ListNodesHandler godoc
//
//	@Summary		List fake SMD nodes
//	@Description	Available only when fake SMD is enabled.
//	@Tags			admin,fake-smd
//	@Produce		json
//	@Success		200	{array}		cistore.OpenCHAMIComponent
//	@Failure		500	{string}	string
//	@Security		BearerAuth
//	@Failure		401	{string}	string	"Missing or invalid bearer JWT"
//	@Failure		503	{string}	string	"Admin JWT verifier unavailable"
//	@Router			/admin/fake-sm/nodes [get]
func ListNodesHandler(f *FakeSMDClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodes := f.ListNodes()
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(nodes); err != nil {
			log.Error().Err(err).Msg("Failed to encode nodes")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// UpdateNodeHandler godoc
//
//	@Summary		Update a fake SMD node
//	@Description	Available only when fake SMD is enabled.
//	@Tags			admin,fake-smd
//	@Accept			json
//	@Param			node	body		OpenCHAMINodeWithGroups	true	"Node data and groups"
//	@Failure		400		{string}	string
//	@Param			id		path		string	true	"Node ID"
//	@Success		200		{object}	nil
//	@Failure		500		{string}	string
//	@Security		BearerAuth
//	@Failure		401	{string}	string	"Missing or invalid bearer JWT"
//	@Failure		503	{string}	string	"Admin JWT verifier unavailable"
//	@Router			/admin/fake-sm/nodes/{id} [put]
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
