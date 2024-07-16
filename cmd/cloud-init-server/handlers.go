package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/internal/memstore"
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	yaml "gopkg.in/yaml.v2"
)

type CiHandler struct {
	store ciStore
	sm    *smdclient.SMDClient
}

func NewCiHandler(s ciStore, c *smdclient.SMDClient) *CiHandler {
	return &CiHandler{
		store: s,
		sm:    c,
	}
}

// ListEntries godoc
// @Summary List all cloud-init entries
// @Description List all cloud-init entries
// @Produce json
// @Success 200 {object} map[string]CI
// @Router /harbor [get]
func (h CiHandler) ListEntries(w http.ResponseWriter, r *http.Request) {
	ci, err := h.store.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	render.JSON(w, r, ci)
}

// AddEntry godoc
// @Summary Add a new cloud-init entry
// @Description Add a new cloud-init entry
// @Accept json
// @Produce json
// @Param ci body CI true "Cloud-init entry to add"
// @Success 200 {string} string "name of the new entry"
// @Failure 400 {string} string "bad request"
// @Failure 500 {string} string "internal server error"
// @Router /harbor [post]
func (h CiHandler) AddEntry(w http.ResponseWriter, r *http.Request) {
	var ci citypes.CI
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err = json.Unmarshal(body, &ci); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.store.Add(ci.Name, ci)
	if err != nil {
		if err == memstore.ExistingEntryErr {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, ci.Name)
}

// GetEntry godoc
// @Summary Get a cloud-init entry
// @Description Get a cloud-init entry
// @Produce json
// @Param id path string true "ID of the cloud-init entry to get"
// @Success 200 {object} CI
// @Failure 404 {string} string "not found"
// @Router /harbor/{id} [get]
func (h CiHandler) GetEntry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ci, err := h.store.Get(id, h.sm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	} else {
		render.JSON(w, r, ci)
	}
}

func (h CiHandler) GetUserData(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ci, err := h.store.Get(id, h.sm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
	ud, err := yaml.Marshal(ci.CIData.UserData)
	if err != nil {
		fmt.Print(err)
	}
	s := fmt.Sprintf("#cloud-config\n%s", string(ud[:]))
	w.Header().Set("Content-Type", "text/yaml")
	w.Write([]byte(s))
}

func (h CiHandler) GetMetaData(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ci, err := h.store.Get(id, h.sm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
	md, err := yaml.Marshal(ci.CIData.MetaData)
	if err != nil {
		fmt.Print(err)
	}
	w.Header().Set("Content-Type", "text/yaml")
	w.Write([]byte(md))
}

func (h CiHandler) GetVendorData(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ci, err := h.store.Get(id, h.sm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
	md, err := yaml.Marshal(ci.CIData.VendorData)
	if err != nil {
		fmt.Print(err)
	}
	w.Header().Set("Content-Type", "text/yaml")
	w.Write([]byte(md))
}

func (h CiHandler) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	var ci citypes.CI
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err = json.Unmarshal(body, &ci); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")

	err = h.store.Update(id, ci)
	if err != nil {
		if err == memstore.NotFoundErr {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, id)
}

func (h CiHandler) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.store.Remove(id)
	if err != nil {
		if err == memstore.NotFoundErr {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, map[string]string{"status": "success"})
}
