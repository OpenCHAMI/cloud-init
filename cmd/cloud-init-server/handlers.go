package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

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

// Enumeration for cloud-init data categories
type ciDataKind uint

// Takes advantage of implicit repetition and iota's auto-incrementing
const (
	UserData ciDataKind = iota
	MetaData
	VendorData
)

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

func (h CiHandler) GetDataByMAC(dataKind ciDataKind) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		// Retrieve the node's xname based on MAC address
		name, err := h.sm.IDfromMAC(id)
		if err != nil {
			log.Print(err)
			name = id // Fall back to using the given name as-is
		} else {
			log.Printf("xname %s with mac %s found\n", name, id)
		}
		// Actually respond with the data
		h.getData(name, dataKind, w)
	}
}

func (h CiHandler) GetDataByIP(dataKind ciDataKind) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Strip port number from RemoteAddr to obtain raw IP
		portIndex := strings.LastIndex(r.RemoteAddr, ":")
		var ip string
		if portIndex > 0 {
			ip = r.RemoteAddr[:portIndex]
		} else {
			ip = r.RemoteAddr
		}
		// Retrieve the node's xname based on IP address
		name, err := h.sm.IDfromIP(ip)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		} else {
			log.Printf("xname %s with ip %s found\n", name, ip)
		}
		// Actually respond with the data
		h.getData(name, dataKind, w)
	}
}

func (h CiHandler) getData(id string, dataKind ciDataKind, w http.ResponseWriter) {
	ci, err := h.store.Get(id, h.sm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}

	var data *map[string]interface{}
	switch dataKind {
	case UserData:
		w.Write([]byte("#cloud-config\n"))
		data = &ci.CIData.UserData
	case MetaData:
		data = &ci.CIData.MetaData
	case VendorData:
		data = &ci.CIData.VendorData
	}

	ydata, err := yaml.Marshal(data)
	if err != nil {
		fmt.Print(err)
	}
	w.Header().Set("Content-Type", "text/yaml")
	w.Write([]byte(ydata))
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

func (h CiHandler) AddGroupData(w http.ResponseWriter, r *http.Request) {
	// type alias to simplify abstraction
	var (
		id   string = chi.URLParam(r, "id")
		body []byte
		data memstore.Data
		err  error
	)

	// read the POST body for JSON data
	body, err = io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// unmarshal data to add to MemStore
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// add a new group to meta-data if it doesn't already exist
	err = h.store.AddGroups(id, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (h CiHandler) GetGroupData(w http.ResponseWriter, r *http.Request) {
	var (
		id    string = chi.URLParam(r, "id")
		data  memstore.Data
		bytes []byte
		err   error
	)

	// get group data from MemStore if it exists
	data, err = h.store.GetGroups(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// marshal to YAML and print the group data to standard output
	bytes, err = yaml.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(bytes)

}

// UpdateGroupData expects a request containing POST data as a JSON. The data
// received in the request should ONLY contain the data to be included for a
// "meta-data.groups" and NOT "meta-data". See "AddGroup" in 'ciMemStore.go'
// for an example.
func (h CiHandler) UpdateGroupData(w http.ResponseWriter, r *http.Request) {
	// type alias to simplify abstraction
	var (
		id   string = chi.URLParam(r, "id")
		body []byte
		data memstore.Data
		err  error
	)

	// read the POST body for JSON data
	body, err = io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// unmarshal data to add to MemStore
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// update groups in meta-data
	err = h.store.UpdateGroups(id, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (h CiHandler) RemoveGroupData(w http.ResponseWriter, r *http.Request) {
	var (
		id  string = chi.URLParam(r, "id")
		err error
	)

	// remove group data with specified name
	err = h.store.RemoveGroups(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func writeInternalError(w http.ResponseWriter, err string) {
	http.Error(w, err, http.StatusInternalServerError)
	// log.Error().Err(err)
}
