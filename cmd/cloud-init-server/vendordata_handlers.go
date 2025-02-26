package main

import (
	"fmt"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// VendorDataHandler godoc
//
//	@Summary		Get vendor data
//	@Description	For OpenCHAMI, the vendor-data will always be a list of other
//	@Description	`#cloud-config` URLs to download and merge.
//	@Description
//	@Description	If the impersonation API is enabled, an ID can be provided in
//	@Description	the URL path using `/admin/impersonation`. In this case, the
//	@Description	vendor-data will be retrieved for the requested ID.
//	@Produce		plain
//	@Success		200	{string}	string
//	@Param			id	path		string	false	"Node ID"
//	@Router			/cloud-init/vendor-data [get]
//	@Router			/cloud-init/admin/impersonation/{id}/vendor-data [get]
func VendorDataHandler(smd smdclient.SMDClientInterface, store cistore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var urlId string = chi.URLParam(r, "id")
		var baseUrl string
		var id = urlId
		var err error
		// If this request includes an id, it can be interrpreted as an impersonation request
		if urlId == "" {
			ip := getActualRequestIP(r)
			// Get the component information from the SMD client
			id, err = smd.IDfromIP(ip)
			if err != nil {
				log.Print(err)
				w.WriteHeader(http.StatusUnprocessableEntity)
				return
			} else {
				log.Debug().Msgf("xname %s with ip %s found\n", id, ip)
			}
		}
		groups, err := smd.GroupMembership(id)
		if err != nil {
			log.Debug().Msgf("Error getting group membership: %s", err)
		}

		clusterDefaults, err := store.GetClusterDefaults()
		if err != nil {
			log.Err(err).Msg("Error getting cluster defaults")
		}
		if clusterDefaults.BaseUrl != "" {
			baseUrl = clusterDefaults.BaseUrl
		}
		extendedInstanceData, err := store.GetInstanceInfo(id)
		if err != nil {
			log.Err(err).Msg("Error getting instance info")
		}
		if extendedInstanceData.CloudInitBaseURL != "" {
			baseUrl = extendedInstanceData.CloudInitBaseURL
		}
		if baseUrl == "" {
			baseUrl = "http://cloud-init:27777"
		}

		payload := "#include\n"
		for _, group_name := range groups {
			payload += fmt.Sprintf("%s/%s.yaml\n", baseUrl, group_name)
		}
		w.Write([]byte(payload))
	}
}
