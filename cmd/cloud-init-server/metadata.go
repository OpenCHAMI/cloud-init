package main

import (
	"fmt"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/rs/zerolog/log"
)

type MetaData struct {
	InstanceID    string       `json:"instance-id" yaml:"instance-id"`
	LocalHostname string       `json:"local-hostname" yaml:"local-hostname" example:"compute-1" description:"Node-specific short hostname"`
	Hostname      string       `json:"hostname" yaml:"hostname" example:"compute-1.demo.openchami.cluster" description:"Node-specific hostname, often FQDN and how other hosts may reference this host"`
	ClusterName   string       `json:"cluster-name" yaml:"cluster-name" example:"demo" description:"Long name of entire cluster, used as a human-readable identifier and is used in the cluster's FQDN"`
	InstanceData  InstanceData `json:"instance-data" yaml:"instance_data"`
}

type InstanceData struct {
	V1 struct {
		CloudName        string      `json:"cloud-name,omitempty" yaml:"cloud_name,omitempty"`
		AvailabilityZone string      `json:"availability-zone,omitempty" yaml:"availability_zone,omitempty"`
		InstanceID       string      `json:"instance-id,omitempty" yaml:"instance_id,omitempty"`
		InstanceType     string      `json:"instance-type,omitempty" yaml:"instance_type,omitempty"`
		LocalHostname    string      `json:"local-hostname,omitempty" yaml:"local_hostname,omitempty"`
		Region           string      `json:"region,omitempty" yaml:"region,omitempty"`
		Hostname         string      `json:"hostname,omitempty" yaml:"hostname,omitempty"`
		LocalIPv4        interface{} `json:"local-ipv4,omitempty" yaml:"local_ipv4,omitempty"`
		CloudProvider    string      `json:"cloud-provider,omitempty" yaml:"cloud_provider,omitempty"`
		PublicKeys       []string    `json:"public-keys,omitempty" yaml:"public_keys,omitempty"`
		VendorData       VendorData  `json:"vendor-data,omitempty" yaml:"vendor_data,omitempty"`
	} `json:"v1" yaml:"v1"`
}

type VendorData struct {
	Version          string           `json:"version" yaml:"version"`
	CloudInitBaseURL string           `json:"cloud-init-base-url,omitempty" yaml:"cloud_init_base_url,omitempty"`
	Rack             string           `json:"rack,omitempty" yaml:"rack,omitempty"`
	Nid              int64            `json:"nid,omitempty" yaml:"nid,omitempty"`
	Role             string           `json:"role,omitempty" yaml:"role,omitempty"`
	SubRole          string           `json:"sub-role,omitempty" yaml:"sub_role,omitempty"`
	Cabinet          string           `json:"cabinet,omitempty" yaml:"cabinet,omitempty"`
	Location         string           `json:"location,omitempty" yaml:"location,omitempty"`
	ClusterName      string           `json:"cluster_name,omitempty" yaml:"cluster_name,omitempty" example:"demo" description:"Long name of entire cluster, used as a human-readable identifier and is used in the cluster's FQDN"`
	Groups           map[string]Group `json:"groups" yaml:"groups" description:"Groups known to cloud-init and their meta-data"`
}

type Group map[string]interface{}

func generateMetaData(component cistore.OpenCHAMIComponent, groups []string, s cistore.Store) MetaData {
	metadata := MetaData{}
	extendedInstanceData, err := s.GetInstanceInfo(component.ID)
	if err != nil {
		log.Err(err).Msg("Error getting instance info")
	}
	// Add in cluster Defaults
	clusterDefaults, err := s.GetClusterDefaults()
	if err != nil {
		log.Err(err).Msg("Error getting cluster defaults")
	}

	// Update extended information from within cloud-init
	metadata.InstanceID = extendedInstanceData.InstanceID
	if extendedInstanceData.LocalHostname == "" {
		metadata.LocalHostname = generateHostname(clusterDefaults.ClusterName, clusterDefaults.ShortName, clusterDefaults.NidLength, component)
	} else {
		metadata.LocalHostname = extendedInstanceData.LocalHostname
	}
	if extendedInstanceData.Hostname == "" {
		metadata.Hostname = generateHostname(clusterDefaults.ClusterName, clusterDefaults.ShortName, clusterDefaults.NidLength, component)
	} else {
		metadata.Hostname = extendedInstanceData.Hostname
	}
	log.Debug().Msgf("Setting ClusterName to %s", clusterDefaults.ClusterName)
	metadata.ClusterName = clusterDefaults.ClusterName

	instanceData := InstanceData{}
	if len(groups) > 0 {
		instanceData.V1.VendorData.Groups = make(map[string]Group)
	}
	for _, group := range groups {
		gd, err := s.GetGroupData(group)
		instanceData.V1.VendorData.Groups[group] = make(map[string]interface{})
		instanceData.V1.VendorData.Groups[group]["Description"] = "No description Found"
		if err != nil {
			log.Print(err)
		} else {
			if gd.Description != "" {
				instanceData.V1.VendorData.Groups[group]["Description"] = gd.Description
			}
			for k, v := range gd.Data {
				instanceData.V1.VendorData.Groups[group][k] = v
			}
		}
	}

	instanceData.V1.LocalIPv4 = component.IP
	instanceData.V1.VendorData.Version = "1.0"

	// Add extended attributes
	instanceData.V1.InstanceID = extendedInstanceData.InstanceID
	instanceData.V1.InstanceType = extendedInstanceData.InstanceType
	if clusterDefaults.BaseUrl != "" {
		instanceData.V1.VendorData.CloudInitBaseURL = clusterDefaults.BaseUrl
	}
	if extendedInstanceData.CloudInitBaseURL != "" {
		instanceData.V1.VendorData.CloudInitBaseURL = extendedInstanceData.CloudInitBaseURL
	}

	instanceData.V1.CloudProvider = clusterDefaults.CloudProvider
	instanceData.V1.VendorData.ClusterName = clusterDefaults.ClusterName
	instanceData.V1.Region = clusterDefaults.Region
	instanceData.V1.AvailabilityZone = clusterDefaults.AvailabilityZone

	// merge cluster defaults and instance specific keys
	instanceData.V1.PublicKeys = append(clusterDefaults.PublicKeys, extendedInstanceData.PublicKeys...)
	metadata.InstanceData = instanceData
	return metadata
}

func generateHostname(clusterName string, shortName string, nidLength int, comp cistore.OpenCHAMIComponent) string {
	// in the future, we might want to map the hostname to an xname or something else.
	nid, _ := comp.NID.Int64()
	var sname string
	var nlen int

	if shortName == "" {
		sname = fmt.Sprintf("%.2s", clusterName)
	} else {
		sname = shortName
	}
	if nidLength == 0 {
		nlen = 4
	} else {
		nlen = nidLength
	}
	log.Debug().Msgf("shortName: %v, nidLength: %v", sname, nlen)
	return fmt.Sprintf("%s%0*d", sname, nlen, nid)
}
