package main

import (
	"crypto/rand"
	"fmt"

	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/rs/zerolog/log"
)

type MetaData struct {
	InstanceID    string       `json:"instance-id" yaml:"instance-id"`
	LocalHostname string       `json:"local-hostname" yaml:"local-hostname"`
	Hostname      string       `json:"hostname" yaml:"hostname"`
	ClusterName   string       `json:"cluster-name" yaml:"cluster-name"`
	InstanceData  InstanceData `json:"instance_data" yaml:"instance_data"`
}

type InstanceData struct {
	V1 struct {
		CloudName        string      `json:"cloud_name,omitempty" yaml:"cloud_name,omitempty"`
		AvailabilityZone string      `json:"availability_zone,omitempty" yaml:"availability_zone,omitempty"`
		InstanceID       string      `json:"instance_id,omitempty" yaml:"instance_id,omitempty"`
		InstanceType     string      `json:"instance_type,omitempty" yaml:"instance_type,omitempty"`
		LocalHostname    string      `json:"local_hostname,omitempty" yaml:"local_hostname,omitempty"`
		Region           string      `json:"region,omitempty" yaml:"region,omitempty"`
		Hostname         string      `json:"hostname,omitempty" yaml:"hostname,omitempty"`
		LocalIPv4        interface{} `json:"local_ipv4,omitempty" yaml:"local_ipv4,omitempty"`
		CloudProvider    string      `json:"cloud_provider,omitempty" yaml:"cloud_provider,omitempty"`
		PublicKeys       []string    `json:"public_keys,omitempty" yaml:"public_keys,omitempty"`
		VendorData       VendorData  `json:"vendor_data,omitempty" yaml:"vendor_data,omitempty"`
	} `json:"v1" yaml:"v1"`
}

type VendorData struct {
	Version          string           `json:"version" yaml:"version"`
	CloudInitBaseURL string           `json:"cloud_init_base_url,omitempty" yaml:"cloud_init_base_url,omitempty"`
	Rack             string           `json:"rack,omitempty" yaml:"rack,omitempty"`
	Nid              int64            `json:"nid,omitempty" yaml:"nid,omitempty"`
	Role             string           `json:"role,omitempty" yaml:"role,omitempty"`
	SubRole          string           `json:"sub_role,omitempty" yaml:"sub_role,omitempty"`
	Cabinet          string           `json:"cabinet,omitempty" yaml:"cabinet,omitempty"`
	Location         string           `json:"location,omitempty" yaml:"location,omitempty"`
	ClusterName      string           `json:"cluster_name,omitempty" yaml:"cluster_name,omitempty"`
	Groups           map[string]Group `json:"groups" yaml:"groups"`
}

type Group map[string]interface{}

func generateInstanceData(component citypes.OpenCHAMIComponent, clusterName string, groups []string, s ciStore) InstanceData {
	cluster_data := InstanceData{}
	if len(groups) > 0 {
		cluster_data.V1.VendorData.Groups = make(map[string]Group)
	}
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
	cluster_data.V1.VendorData.CloudInitBaseURL = "http://cloud-init:27777/cloud-init"
	cluster_data.V1.VendorData.Version = "1.0"
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
