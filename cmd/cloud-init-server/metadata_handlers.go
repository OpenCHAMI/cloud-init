package main

import (
	"net/http"
	"strings"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

func getActualRequestIP(r *http.Request) string {
	var ip string
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// If X-Forwarded-For exists, use the first IP in the list
		ip = strings.Split(xff, ",")[0]
	} else {
		// Otherwise, use the remote address
		portIndex := strings.LastIndex(r.RemoteAddr, ":")
		if portIndex > 0 {
			ip = r.RemoteAddr[:portIndex]
		} else {
			ip = r.RemoteAddr
		}
	}
	return strings.TrimSpace(ip)
}

// MetaDataHandler godoc
//	@Summary		Get meta-data for requesting node
//	@Description	Get meta-data for requesting node based on the requesting IP.
//	@Description
//	@Description	If the impersonation API is enabled, an ID can be provided in
//	@Description	the URL path using `/admin/impersonation`. In this case, the
//	@Description	meta-data will be retrieved for the requested ID.
//	@Produce		application/x-yaml
//	@Success		200	{object}	MetaData
//	@Failure		404	{object}	nil
//	@Failure		422	{object}	nil
//	@Failure		500	{object}	nil
//	@Param			id	path		string	false	"Node ID"
//	@Router			/cloud-init/meta-data [get]
//	@Router			/cloud-init/admin/impersonation/{id}/meta-data [get]
func MetaDataHandler(smd smdclient.SMDClientInterface, store cistore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var urlId string = chi.URLParam(r, "id")
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
				log.Printf("xname %s with ip %s found\n", id, ip)
			}
		}
		log.Debug().Msgf("Getting metadata for id: %s", id)
		smdComponent, err := smd.ComponentInformation(id)
		if err != nil {
			log.Debug().Msgf("Failed to get component information for %s: %s", id, err)
			// If the component information is not available, return a 404
			http.Error(w, "Node not found in SMD. Instance-data not available", http.StatusNotFound)
			return
		}
		groups, err := smd.GroupMembership(id)
		if err != nil {
			// If the group information is not available, return an empty list
			groups = []string{}
		}
		bootIP, err := smd.IPfromID(id)
		if err != nil {
			// If the IP information is not available, return an empty string
			bootIP = ""
		}
		bootMAC, err := smd.MACfromID(id)
		if err != nil {
			// If the MAC information is not available, return an empty string
			bootMAC = ""
		}
		component := cistore.OpenCHAMIComponent{
			Component: smdComponent,
			IP:        bootIP,
			MAC:       bootMAC,
		}

		metadata := generateMetaData(component, groups, store)

		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		yamlData, err := yaml.Marshal(metadata)
		if err != nil {
			http.Error(w, "Failed to encode metadata to YAML", http.StatusInternalServerError)
			return
		}
		w.Write(yamlData)
	}
}
