package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
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

func InstanceDataHandler(smd smdclient.SMDClientInterface, clusterName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ip should be the ip address that the request originated from.  Check the headers to see if the request has been forwarded and the remote IP is preserved
		// Check for the first ip in the X-Forwarded-For header if it exists
		var ip string
		if r.Header.Get("X-Forwarded-For") != "" {
			// If it exists, use that
			ip = strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]
		} else {
			portIndex := strings.LastIndex(r.RemoteAddr, ":")
			if portIndex > 0 {
				ip = r.RemoteAddr[:portIndex]
			} else {
				ip = r.RemoteAddr
			}
		}
		// Get the component information from the SMD client
		id, err := smd.IDfromIP(ip)
		if err != nil {
			fmt.Print(err)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		} else {
			fmt.Printf("xname %s with ip %s found\n", id, ip)
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

		component := citypes.OpenCHAMIComponent{
			Component:   smdComponent,
			ClusterName: clusterName,
		}

		cluster_data := generateInstanceData(component, groups)
		// Return the instance data as json
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cluster_data)
	}
}

func generateInstanceData(component citypes.OpenCHAMIComponent, groups []string) citypes.ClusterData {
	cluster_data := citypes.ClusterData{}
	cluster_data.InstanceData.V1.CloudName = "OpenCHAMI"
	cluster_data.InstanceData.V1.AvailabilityZone = "lanl-yellow"
	cluster_data.InstanceData.V1.InstanceID = generateInstanceId()
	cluster_data.InstanceData.V1.InstanceType = "t2.micro"
	cluster_data.InstanceData.V1.LocalHostname = genHostname(component.ClusterName, component)
	cluster_data.InstanceData.V1.Region = "us-west"
	cluster_data.InstanceData.V1.Hostname = genHostname(component.ClusterName, component)
	cluster_data.InstanceData.V1.LocalIPv4 = component.IP
	cluster_data.InstanceData.V1.CloudProvider = "OpenCHAMI"
	cluster_data.InstanceData.V1.PublicKeys = []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD..."}
	cluster_data.InstanceData.V1.VendorData.Version = "1.0"
	for _, group := range groups {
		cluster_data.InstanceData.V1.VendorData.Groups = append(cluster_data.InstanceData.V1.VendorData.Groups, struct {
			GroupName string            "json:\"group_name\""
			Metadata  map[string]string "json:\"metadata,omitempty\""
		}{GroupName: group})
	}
	return cluster_data
}

func genHostname(clusterName string, comp citypes.OpenCHAMIComponent) string {
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
