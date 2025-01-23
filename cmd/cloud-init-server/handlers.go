package main

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/OpenCHAMI/cloud-init/pkg/wgtunnel"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
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

// Phone home should be a POST request x-www-form-urlencoded like this: pub_key_rsa=rsa_contents&pub_key_ecdsa=ecdsa_contents&pub_key_ed25519=ed25519_contents&instance_id=i-87018aed&hostname=myhost&fqdn=myhost.internal
func PhoneHomeHandler(store cistore.Store, wg *wgtunnel.InterfaceManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		ip := getActualRequestIP(r)
		log.Info().Msgf("Phone home request from %s", ip)
		// TODO: validate the request IP against the SMD client and reject if needed

		err := r.ParseForm()
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
				wg.RemovePeer(ip)
			}()

			w.WriteHeader(http.StatusOK)
		}
	}
}
