package bss

import (
	"encoding/json"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/pkg/smdclient"
	"github.com/google/uuid"
)

// V1AddBootParamsHandler adds a new boot params to the store
// If a list of nodes is provided, we need to store this the old way
// by normalizing the list of nodes to a list of ids and storing the same data for each node
// If a group is provided, we can store it the new way
func V1AddBootParamsHandler(store Store, smd *smdclient.SMDClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// convert request body to V1BootParams
		var v1BootParams BootParamsV1
		err := json.NewDecoder(r.Body).Decode(&v1BootParams)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		//normalize the list of nodes to a list of ids (xnames)
		var xnames []string
		var badmacs []string
		if v1BootParams.Hosts != nil {
			xnames = append(xnames, v1BootParams.Hosts...)
		}
		if v1BootParams.MACs != nil {
			for _, mac := range v1BootParams.MACs {
				xname, err := smd.IDfromMAC(mac)
				if err != nil {
					badmacs = append(badmacs, mac)
				} else {
					xnames = append(xnames, xname)
				}
			}
		}
		if v1BootParams.NIDs != nil {
			http.Error(w, "NIDs not supported", http.StatusBadRequest)
			return
		}
		if len(xnames) != 0 {
			for _, xname := range xnames {
				// We need a referral token for each node
				// Generating one for now, but we should use a better mechanism in the future
				referralToken := uuid.New().String()
				v1BootParams.ReferralToken = referralToken
				store.SetV1(xname, &v1BootParams)
			}
		}
		// return 201 created with a BSS-Referral-Token header
		referralToken := uuid.New().String()
		w.Header().Set("BSS-Referral-Token", referralToken)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(
			struct {
				Hosts   []string `json:"hosts"`
				BadMACs []string `json:"bad_macs"`
			}{
				Hosts:   xnames,
				BadMACs: badmacs,
			},
		)
	}
}
