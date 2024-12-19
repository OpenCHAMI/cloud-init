package main

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
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
	var (
		body []byte
		err  error
		data cistore.GroupData
	)

	// read the POST body for JSON data
	body, err = io.ReadAll(r.Body)
	if err != nil {
		return data, err
	}
	// unmarshal data to add to group data
	err = json.Unmarshal(body, &data)
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
