package main

import (
	"net/http"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func UserDataHandler(w http.ResponseWriter, r *http.Request) {
	payload := `#cloud-config`
	w.Write([]byte(payload))
}

func GroupUserDataHandler(smd smdclient.SMDClientInterface, store cistore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, group, err := getIDAndGroup(r, smd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if !isUserInGroup(id, group, smd) {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		data, err := store.GetGroupData(group)
		if err != nil {
			log.Err(err).Msgf("No information stored for group %s. returning an empty #cloud-config", group)
			w.Write([]byte("#cloud-config"))
			return
		}

		w.Write(data.File.Content)
	}
}

func getIDAndGroup(r *http.Request, smd smdclient.SMDClientInterface) (string, string, error) {
	id := chi.URLParam(r, "id")
	group := chi.URLParam(r, "group")

	if id == "" {
		ip := getActualRequestIP(r)
		var err error
		id, err = smd.IDfromIP(ip)
		if err != nil {
			return "", "", err
		}
	}

	return id, group, nil
}

func isUserInGroup(id, group string, smd smdclient.SMDClientInterface) bool {
	groups, err := smd.GroupMembership(id)
	if err != nil {
		log.Debug().Msg(err.Error())
		// If the group information is not available, return an empty list
		groups = []string{}
	}

	return contains(groups, group)
}

func contains(list []string, item string) bool {
	for _, g := range list {
		if g == item {
			return true
		}
	}
	return false
}
