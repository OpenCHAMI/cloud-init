package main

import (
	"encoding/json"
	"io"
	"net/http"

	// Import to run swag.Register() to generated docs
	_ "github.com/OpenCHAMI/cloud-init/docs"
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/OpenCHAMI/cloud-init/pkg/wgtunnel"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/swaggo/swag"
)

type CiHandler struct {
	store       cistore.Store
	sm          smdclient.SMDClientInterface
	clusterName string
}

func NewCiHandler(s cistore.Store, c smdclient.SMDClientInterface, clusterName string) *CiHandler {
	return &CiHandler{
		store:       s,
		sm:          c,
		clusterName: clusterName,
	}
}

func parseData(r *http.Request) (cistore.GroupData, error) {
	var data cistore.GroupData

	// Read the POST body for JSON data
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return data, err
	}

	// Use the GroupData method to parse the JSON
	err = data.ParseFromJSON(body)
	if err != nil {
		log.Debug().Msgf("Error unmarshalling JSON data: %v", err)
		return data, err
	}
	return data, nil
}

// DocsHandler godoc
//
//	@Summary	Return JSON-formatted OpenAPI documentation
//	@Produce	json
//	@Success	200	{object}	string
//	@Failure	500	{object}	nil
//	@Router		/cloud-init/openapi.json [get]
func DocsHandler(w http.ResponseWriter, r *http.Request) {
	doc, err := swag.ReadDoc()
	if err != nil {
		log.Error().Msgf("Error reading OpenAPI docs: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	bDoc := []byte(doc)
	w.Write(bDoc)
}

// SetClusterDataHandler godoc
//
//	@Summary		Set cluster defaults
//	@Description	Set default meta-data values for cluster.
//	@Tags			admin,cluster-defaults
//	@Accept			json
//	@Success		201		{object}	nil
//	@Failure		400		{object}	nil
//	@Failure		500		{object}	nil
//	@Param			data	body		cistore.ClusterDefaults	true	"Cluster defaults data"
//	@Router			/cloud-init/admin/cluster-defaults [post]
func SetClusterDataHandler(store cistore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		data := cistore.ClusterDefaults{}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error().Msgf("Error reading request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// unmarshal data to add to group data
		err = json.Unmarshal(body, &data)
		if err != nil {
			log.Debug().Msgf("Error unmarshalling JSON data: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = store.SetClusterDefaults(data)
		if err != nil {
			log.Error().Msgf("Error setting cluster defaults: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// GetClusterDataHandler godoc
//
//	@Summary		Get cluster defaults
//	@Description	Get default meta-data values for cluster.
//	@Tags			admin,cluster-defaults
//	@Produce		json
//	@Success		200	{object}	cistore.ClusterDefaults
//	@Failure		500	{object}	nil
//	@Router			/cloud-init/admin/cluster-defaults [get]
func GetClusterDataHandler(store cistore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := store.GetClusterDefaults()
		if err != nil {
			log.Error().Msgf("Error getting cluster defaults: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Error().Msgf("Error marshalling cluster defaults: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
	}
}

// InstanceInfoHandler godoc
//
//	@Summary		Set node-specific meta-data
//	@Description	Set meta-data for a specific node ID, overwriting relevant group meta-data.
//	@Tags			admin,instance-data
//	@Accept			json
//	@Success		201				{object}	nil
//	@Failure		400				{object}	nil
//	@Failure		500				{object}	nil
//	@Param			id				path		string							true	"Node ID"
//	@Param			instance-info	body		cistore.OpenCHAMIInstanceInfo	true	"Instance info data"
//	@Router			/cloud-init/admin/instance-info/{id} [put]
func InstanceInfoHandler(sm smdclient.SMDClientInterface, store cistore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var id string = chi.URLParam(r, "id")
		var info cistore.OpenCHAMIInstanceInfo
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error().Msgf("Error reading request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// unmarshal data to add to group data
		err = json.Unmarshal(body, &info)
		if err != nil {
			log.Debug().Msgf("Error unmarshalling JSON data: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = store.SetInstanceInfo(id, info)
		if err != nil {
			log.Error().Msgf("Error setting instance info: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// PhoneHomeHandler godoc
//
//	@Summary		Signal to cloud-init server that host has completed running cloud-init configuration
//	@Description	Signal to the cloud-init server that the specific host has completed running
//	@Description	the cloud-init configuration tasks so that, if a WireGuard tunnel is being used,
//	@Description	it can be torn down. This endpoint should not be manually requested by a user
//	@Description	but is only meant to be used by a cloud-init client that has received its
//	@Description	config from an OpenCHAMI cloud-init server.
//	@Tags			phone-home
//	@Success		200				{object}	nil
//	@Failure		400				{object}	nil
//	@Param			id				path		string	true	"Node's unique identifier"
//	@Param			pub_key_rsa		formData	string	true	"Node's WireGuard RSA public key"
//	@Param			pub_key_ecdsa	formData	string	true	"Node's WireGuard ECDSA public key"
//	@Param			pub_key_ed25519	formData	string	true	"Node's WireGuard ED35519 public key"
//	@Param			instance_id		formData	string	true	"Node's given instance ID"
//	@Param			hostname		formData	string	true	"Node's given hostname"
//	@Param			fqdn			formData	string	true	"Node's given fully-qualified domain name"
//	@Router			/cloud-init/phone-home/{id} [post]
func PhoneHomeHandler(wg *wgtunnel.InterfaceManager, sm smdclient.SMDClientInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		ip := getActualRequestIP(r)
		log.Info().Msgf("Phone home request from %s", ip)
		// TODO: validate the request IP against the SMD client and reject if needed

		id, err := sm.IDfromIP(ip)
		if err != nil {
			log.Error().Msgf("Error getting ID from IP: %v", err)
		}
		peerName, err := sm.IPfromID(id)
		if err != nil {
			log.Error().Msgf("Error getting IP from ID: %v", err)
		}
		err = r.ParseForm()
		if err != nil {
			log.Error().Msgf("Error parsing form data: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		pubKeyRsa := r.FormValue("pub_key_rsa")
		pubKeyEcdsa := r.FormValue("pub_key_ecdsa")
		pubKeyEd25519 := r.FormValue("pub_key_ed25519")
		instanceId := r.FormValue("instance_id")
		hostname := r.FormValue("hostname")
		fqdn := r.FormValue("fqdn")

		log.Info().
			Str("pub_key_rsa", pubKeyRsa).
			Str("pub_key_ecdsa", pubKeyEcdsa).
			Str("pub_key_ed25519", pubKeyEd25519).
			Str("instance_id", instanceId).
			Str("hostname", hostname).
			Str("fqdn", fqdn).
			Msgf("Received phone home data: pub_key_rsa=%s, pub_key_ecdsa=%s, pub_key_ed25519=%s, instance_id=%s, hostname=%s, fqdn=%s",
				pubKeyRsa, pubKeyEcdsa, pubKeyEd25519, instanceId, hostname, fqdn)

		if wg != nil {
			go func() {
				wg.RemovePeer(peerName)
			}()

			w.WriteHeader(http.StatusOK)
		}
	}
}
