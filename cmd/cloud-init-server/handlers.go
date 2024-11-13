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
		if err == memstore.ExistingErr {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, ci.Name)
}

// AddUserEntry godoc
// @Summary Add a new user-data entry in specified cloud-init data
// @Description Add a new user-data entry in specified cloud-init data
// @Accept json
// @Produce json
// @Param ci body CI true "User-ata entry to add to cloud-init data"
// @Success 200 {string} string "name of the new entry"
// @Failure 400 {string} string "bad request"
// @Failure 500 {string} string "internal server error"
// @Router /harbor [post]
func (h CiHandler) AddUserEntry(w http.ResponseWriter, r *http.Request) {
	var (
		ci       citypes.CI
		userdata citypes.UserData
		body     []byte
		err      error
	)

	// read the request body for user data
	body, err = io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// unmarshal only to user data and not cloud-init data
	if err = json.Unmarshal(body, &userdata); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// store the userdata in the cloud-init data
	ci.CIData.UserData = userdata

	// add the cloud-init data
	err = h.store.Add(ci.Name, ci)
	if err != nil {
		if err == memstore.ExistingErr {
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

func (h CiHandler) UpdateUserEntry(w http.ResponseWriter, r *http.Request) {
	var (
		id       = chi.URLParam(r, "id")
		ci       citypes.CI
		userdata citypes.UserData
		err      error
	)

	// read the request body for user data
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// unmarshal only to user data and not cloud-init data
	if err = json.Unmarshal(body, &userdata); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// set the user-data to overwrite the existing entry
	ci.CIData.UserData = userdata
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

func (h CiHandler) GetGroups(w http.ResponseWriter, r *http.Request) {
	var (
		groups map[string]citypes.Group
		bytes  []byte
		err    error
	)
	groups = h.store.GetGroups()
	bytes, err = json.MarshalIndent(groups, "", "\t")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(bytes)
}

func (h CiHandler) AddGroupData(w http.ResponseWriter, r *http.Request) {
	var (
		id   string = chi.URLParam(r, "id")
		data citypes.GroupData
		err  error
	)

	data, err = parseData(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.store.AddGroupData(id, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (h CiHandler) GetGroupData(w http.ResponseWriter, r *http.Request) {
	var (
		id    string = chi.URLParam(r, "id")
		data  citypes.GroupData
		bytes []byte
		err   error
	)

	data, err = h.store.GetGroupData(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bytes, err = yaml.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(bytes)
}
func (h CiHandler) UpdateGroupData(w http.ResponseWriter, r *http.Request) {
	var (
		id   string = chi.URLParam(r, "id")
		data citypes.GroupData
		err  error
	)

	data, err = parseData(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// update group key-value data
	err = h.store.UpdateGroupData(id, data)
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
	err = h.store.RemoveGroupData(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func writeInternalError(w http.ResponseWriter, err string) {
	http.Error(w, err, http.StatusInternalServerError)
	// log.Error().Err(err)
}

func parseData(w http.ResponseWriter, r *http.Request) (citypes.GroupData, error) {
	var (
		body []byte
		err  error
		data citypes.GroupData
	)

	// read the POST body for JSON data
	body, err = io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	// unmarshal data to add to group data
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
