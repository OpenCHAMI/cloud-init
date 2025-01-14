package main

import (
	"encoding/json"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/go-chi/chi/v5"
	yaml "gopkg.in/yaml.v2"
)

func (h CiHandler) GetGroups(w http.ResponseWriter, r *http.Request) {
	var (
		groups map[string]cistore.GroupData
		bytes  []byte
		err    error
	)
	groups = h.store.GetGroups()
	bytes, err = json.MarshalIndent(groups, "", "\t")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(bytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

/*
AddGroupHandler adds a new group with it's associated data specified by the user.

*/
// AddGroupHandler handles the HTTP request for adding a new group.
// It parses the request data into a GroupData struct, validates it,
// and then attempts to store it using the handler's store. If successful,
// it sets the Location header to the new group's URL and responds with
// HTTP status 201 Created. If there is an error during parsing or storing,
// it responds with the appropriate HTTP error status.
//
// Curl Example:
//
// curl -X POST http://localhost:27777/cloud-init/admin/groups/ \
//      -H "Content-Type: application/json" \
//      -d '{
//           "name": "x3000",
//           "description": "Cabinet x3000",
//           "data": {
//             "syslog_aggregator": "192.168.0.1"
//            },
//           "file": {
//             "content": "#cloud-config\nrsyslog:\n  remotes: {x3000: \"192.168.0.5\"}\nservice_reload_command: auto\n",
//             "encoding": "plain"
//           }
//         }'
// It parses the request data into a GroupData struct and attempts to add it to the store.
// Encoding options are "plain" or "base64".
// If parsing fails, it responds with a 422 Unprocessable Entity status.
// If adding the group data to the store fails, it responds with a 409 Conflict status.
// On success, it sets the Location header to the new group's URL and responds with a 201 Created status.
func (h CiHandler) AddGroupHandler(w http.ResponseWriter, r *http.Request) {
	var (
		data cistore.GroupData
		err  error
	)

	data, err = parseData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	err = h.store.AddGroupData(data.Name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Location", "/groups/"+data.Name)
	w.WriteHeader(http.StatusCreated)

}

func (h CiHandler) GetGroupHandler(w http.ResponseWriter, r *http.Request) {
	var (
		id    string = chi.URLParam(r, "id")
		data  cistore.GroupData
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

func (h CiHandler) UpdateGroupHandler(w http.ResponseWriter, r *http.Request) {
	var (
		groupName string = chi.URLParam(r, "name")
		data      cistore.GroupData
		err       error
	)

	data, err = parseData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// update group key-value data
	err = h.store.UpdateGroupData(groupName, data, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", "/groups/"+data.Name)
	w.WriteHeader(http.StatusCreated)
}

func (h CiHandler) RemoveGroupHandler(w http.ResponseWriter, r *http.Request) {
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
