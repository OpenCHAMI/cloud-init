package citypes

import (
	base "github.com/Cray-HPE/hms-base"
)

type CI struct {
	Name   string `json:"name"`
	CIData CIData `json:"cloud-init"`
}

type CIData struct {
	UserData   map[string]any `json:"user-data"`
	MetaData   map[string]any `json:"meta-data"`
	VendorData map[string]any `json:"vendor-data"`
}

type (
	// only defined for readibility
	UserData = map[string]any
)

type WriteFiles struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Group   string `json:"group,omitempty"`
}

type GroupData struct {
	Name    string         `json:"name,omitempty"`
	Data    MetaDataKV     `json:"meta-data,omitempty"`
	Actions map[string]any `json:"user-data,omitempty"`
}

type MetaDataKV map[string]string // Metadata for the group may only contain key value pairs

type ClusterData struct {
	InstanceData struct {
		V1 struct {
			CloudName        string      `json:"cloud_name"`
			AvailabilityZone string      `json:"availability_zone"`
			InstanceID       string      `json:"instance_id"`
			InstanceType     string      `json:"instance_type"`
			LocalHostname    string      `json:"local_hostname"`
			Region           string      `json:"region"`
			Hostname         string      `json:"hostname"`
			LocalIPv4        interface{} `json:"local_ipv4"`
			CloudProvider    string      `json:"cloud_provider"`
			PublicKeys       []string    `json:"public_keys"`
			VendorData       struct {
				Version string `json:"version"`
				Groups  []struct {
					GroupName string            `json:"group_name"`
					Metadata  map[string]string `json:"metadata,omitempty"`
				} `json:"groups"`
			} `json:"vendor_data"`
		} `json:"v1"`
	} `json:"instance-data"`
}

type OpenCHAMIComponent struct {
	base.Component
	ClusterName string `json:"cluster_name"`
	MAC         string `json:"mac"`
	IP          string `json:"ip"`
}
