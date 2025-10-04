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
//	@Router			/vendor-data [get]
//	@Router			/admin/impersonation/{id}/vendor-data [get]
func VendorDataHandler(smd smdclient.SMDClientInterface, store cistore.Store, baseUrl string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlId := chi.URLParam(r, "id")
		var id = urlId
		var err error
		// If this request includes an id, it can be interrpreted as an impersonation request
		if urlId == "" {
			log.Debug().Msg("no id specified in request, attempting to identify based on requesting IP")
			ip := getActualRequestIP(r)
			log.Debug().Msgf("requesting IP is: %s", ip)
			// Get the component information from the SMD client
			id, err = smd.IDfromIP(ip)
			if err != nil {
				log.Printf("did not find id from ip %s: %v", ip, err)
				w.WriteHeader(http.StatusUnprocessableEntity)
				return
			} else {
				log.Debug().Msgf("xname %s with ip %s found\n", id, ip)
			}
		}
		groups, err := smd.GroupMembership(id)
		if err != nil {
			if esr, ok := err.(smdclient.ErrSMDResponse); ok {
				switch esr.HTTPResponse.StatusCode {
				case http.StatusNotFound:
					log.Warn().Msgf("node %s not found in SMD, include list will be empty", id)
				case http.StatusBadRequest:
					log.Warn().Msgf("node %s is an invalid xname in SMD, include list will be empty", id)
				default:
					log.Error().Err(err).Msg("unhandled HTTP response from SMD, include list will be empty")
				}
			} else {
				log.Error().Err(err).Msgf("failed to get group membership for id %s, include list will be empty", id)
			}
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
			log.Err(err).Msgf("Error getting instance info for id %s", id)
		}
		if extendedInstanceData.CloudInitBaseURL != "" {
			baseUrl = extendedInstanceData.CloudInitBaseURL
		}

		payload := "#include\n"
		for _, group_name := range groups {
			payload += fmt.Sprintf("%s/%s.yaml\n", baseUrl, group_name)
		}
		if _, err = w.Write([]byte(payload)); err != nil {
			log.Error().Err(err).Msg("failed to write response")
		}
	}
}
