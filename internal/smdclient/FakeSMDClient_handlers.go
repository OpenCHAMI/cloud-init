package smdclient

import (
	"encoding/json"
	"net/http"

	base "github.com/Cray-HPE/hms-base"
)

type AddNodeStruct struct {
	base.Component
	BootMAC       string `json:"boot-mac,omitempty"`
	BootIPAddress string `json:"boot-ip,omitempty"`
}

func AddNodeToInventoryHandler(f *FakeSMDClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var addNode AddNodeStruct
		err := json.NewDecoder(r.Body).Decode(&addNode)
		if err != nil {
			if err, _ := f.AddNodeToInventory(addNode.Component, addNode.BootMAC, addNode.BootIPAddress); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Location", r.URL.Path+"/"+addNode.ID)

		}
	}
}
