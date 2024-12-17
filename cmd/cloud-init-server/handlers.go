package main

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/rs/zerolog/log"
)

type CiHandler struct {
	store       ciStore
	sm          smdclient.SMDClientInterface
	clusterName string
}

func NewCiHandler(s ciStore, c smdclient.SMDClientInterface, clusterName string) *CiHandler {
	return &CiHandler{
		store:       s,
		sm:          c,
		clusterName: clusterName,
	}
}

func parseData(r *http.Request) (citypes.GroupData, error) {
	var (
		body []byte
		err  error
		data citypes.GroupData
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
