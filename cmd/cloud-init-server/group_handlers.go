package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

func (h CiHandler) GetGroups(w http.ResponseWriter, r *http.Request) {
	var (
		groups map[string]citypes.GroupData
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

func (h CiHandler) AddGroupHandler(w http.ResponseWriter, r *http.Request) {
	var (
		data citypes.GroupData
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

func (h CiHandler) UpdateGroupHandler(w http.ResponseWriter, r *http.Request) {
	var (
		groupName string = chi.URLParam(r, "name")
		data      citypes.GroupData
		err       error
	)

	data, err = parseData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// update group key-value data
	err = h.store.UpdateGroupData(groupName, data)
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

func GroupUserDataHandler(smd smdclient.SMDClientInterface, store ciStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			id    string = chi.URLParam(r, "id")
			group string = chi.URLParam(r, "group")
			data  citypes.GroupData
			bytes []byte
			err   error
		)
		if id == "" {

			ip := getActualRequestIP(r)
			id, err = smd.IDfromIP(ip)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		groups, err := smd.GroupMembership(id)
		if err != nil {
			log.Debug().Msg(err.Error())
			// If the group information is not available, return an empty list
			groups = []string{}
		}
		if !contains(groups, group) {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		data, err = store.GetGroupData(group)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Debug().Msgf("GroupUserDataHandler: %v", data)
		if data.File.Encoding == "base64" {
			bytes, err = base64.StdEncoding.DecodeString(string(data.File.Content))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			bytes = []byte(data.File.Content)
		}
		w.Write(bytes)
	}
}

func contains(list []string, item string) bool {
	for _, g := range list {
		if g == item {
			return true
		}
	}
	return false
}
