package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// The instance-data endpoint should return information about the instance.
// For information about the standard items, check the docs at https://cloudinit.readthedocs.io/en/latest/explanation/instancedata.html#standardised-instance-data-json-v1-keys
// It should be accessible at /latest/instance-data as yaml and /latest/instance-data/json as json.
// /latest/instance-data should obey Accept headers and return the appropriate format as well.
// The instance-data endpoint should return a 404 if the instance data is not available.

/* The payload we're targeting here is:

#cloud-config
instance-data:
  v1:
    cloud_name: ""
    availability_zone: ""
    instance_id: ""
    instance_type: ""
    local_hostname: ""
    region: ""
    hostname: ""
    local_ipv4: null
    cloud_provider: ""
    public_keys:
      - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD...
    vendor_data:
      version: 1.0
	    groups:
		  - group_name
		    key: value
			key2: value2
*/

func InstanceDataHandler(smd smdclient.SMDClientInterface, store ciStore, clusterName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var id string = chi.URLParam(r, "id")
		var err error
		if id == "" {
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
		smdComponent, err := smd.ComponentInformation(id)
		if err != nil {
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
		component := citypes.OpenCHAMIComponent{
			Component: smdComponent,
			IP:        bootIP,
			MAC:       bootMAC,
		}

		cluster_data := generateInstanceData(component, clusterName, groups, store)
		// Return the instance data as json
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cluster_data)
	}
}

func generateInstanceData(component citypes.OpenCHAMIComponent, clusterName string, groups []string, s ciStore) citypes.InstanceData {
	cluster_data := citypes.InstanceData{}
	cluster_data.V1.CloudName = "OpenCHAMI"
	cluster_data.V1.AvailabilityZone = "lanl-yellow"
	cluster_data.V1.InstanceID = generateInstanceId()
	cluster_data.V1.InstanceType = "t2.micro"
	cluster_data.V1.LocalHostname = generateHostname(clusterName, component)
	cluster_data.V1.Region = "us-west"
	cluster_data.V1.Hostname = generateHostname(clusterName, component)
	cluster_data.V1.LocalIPv4 = component.IP
	cluster_data.V1.CloudProvider = "OpenCHAMI"
	cluster_data.V1.PublicKeys = []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD..."}
	cluster_data.V1.VendorData.Version = "1.0"
	cluster_data.V1.VendorData.Groups = make(map[string]citypes.Group)
	for _, group := range groups {
		gd, err := s.GetGroupData(group)
		cluster_data.V1.VendorData.Groups[group] = make(map[string]interface{})
		cluster_data.V1.VendorData.Groups[group]["Description"] = "No description Found"
		if err != nil {
			log.Print(err)
		} else {
			if gd.Description != "" {
				cluster_data.V1.VendorData.Groups[group]["Description"] = gd.Description
			}
			for k, v := range gd.Data {
				cluster_data.V1.VendorData.Groups[group][k] = v
			}
		}
	}
	return cluster_data
}

func generateHostname(clusterName string, comp citypes.OpenCHAMIComponent) string {
	// in the future, we might want to map the hostname to an xname or something else.
	switch comp.Role {
	case "compute":
		nid, _ := comp.NID.Int64()
		return fmt.Sprintf("%.2s%04d", clusterName, nid)
	case "io":
		nid, _ := comp.NID.Int64()
		return fmt.Sprintf("%.2s-io%02d", clusterName, nid)
	case "front_end":
		nid, _ := comp.NID.Int64()
		return fmt.Sprintf("%.2s-fe%02d", clusterName, nid)
	default:
		nid, _ := comp.NID.Int64()
		return fmt.Sprintf("%.2s%04d", clusterName, nid)
	}
}

func generateInstanceId() string {
	// in the future, we might want to map the instance-id to an xname or something else.
	return generateUniqueID("i")

}

func generateUniqueID(prefix string) string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", prefix, b)
}
