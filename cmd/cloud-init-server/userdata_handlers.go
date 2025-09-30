package main

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// UserDataHandler godoc
//
//	@Summary		Get user-data for requesting node
//	@Description	Get user-data for requesting node base on the requesting IP. For
//	@Description	OpenCHAMI, this will always be `#cloud-config`.
//	@Description
//	@Description	If the impersonation API is enabled, an ID can be provided in
//	@Description	the URL path using `/admin/impersonation`. In this case, the
//	@Description	user-data will be retrieved for the requested ID.
//	@Produce		plain
//	@Success		200	{object}	string
//	@Param			id	path		string	false	"Node ID"
//	@Router			/user-data [get]
//	@Router			/admin/impersonation/{id}/user-data [get]
func UserDataHandler(w http.ResponseWriter, r *http.Request) {
	payload := `#cloud-config`
	if _, err := w.Write([]byte(payload)); err != nil {
		log.Error().Err(err).Msg("failed to write response")
	}
}

// GroupUserDataHandler godoc
//
//	@Summary		Get user-data for a particular group
//	@Description	Get user-data for a particular group based on its name.
//	@Description
//	@Description	If the impersonation API is enabled, an ID can be provided in
//	@Description	the URL path using `/admin/impersonation`. In this case, the
//	@Description	group user-data will be retrieved for the requested ID.
//	@Produce		plain
//	@Success		200		{object}	string
//	@Failure		404		{object}	nil
//	@Failure		500		{object}	nil
//	@Param			id		path		string	false	"Node ID"
//	@Param			group	path		string	true	"Group name"
//	@Router			/{group}.yaml [get]
//	@Router			/admin/impersonation/{id}/{group}.yaml [get]
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
			if _, err2 := w.Write([]byte("#cloud-config")); err2 != nil {
				log.Error().Err(err).Msg("failed to write response")
			}
			return
		}

		// Make sure cloud-config content is plaintext before returning
		if data.File.Encoding == "base64" {
			decodedContent, err := base64.StdEncoding.DecodeString(string(data.File.Content))
			if err != nil {
				newErr := fmt.Errorf("failed to base64-decode cloud-config: %w", err)
				http.Error(w, newErr.Error(), http.StatusInternalServerError)
				return
			}
			data.File.Content = decodedContent
			data.File.Encoding = "plain"
		}

		if _, err = w.Write(data.File.Content); err != nil {
			log.Error().Err(err).Msg("failed to write response")
		}
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
