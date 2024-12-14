package citypes

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	base "github.com/Cray-HPE/hms-base"
)

type CI struct {
	Name   string `json:"name"`
	CIData CIData `json:"cloud-init"`
}

type CIData struct {
	MetaData map[string]any `json:"meta-data"`
}

type CloudConfigFile struct {
	Content  []byte `json:"content"`
	Name     string `json:"filename"`
	Encoding string `json:"encoding,omitempty"` // base64 or plain
}

// Custom unmarshaler for CloudConfigFile
func (f *CloudConfigFile) UnmarshalJSON(data []byte) error {
	// Define a helper struct to match the JSON structure
	type Alias CloudConfigFile
	aux := &struct {
		Content string `json:"content"` // Temporarily hold content as a string
		*Alias
	}{
		Alias: (*Alias)(f),
	}

	// Unmarshal into the helper struct
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle encoding
	switch aux.Encoding {
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(aux.Content)
		if err != nil {
			return fmt.Errorf("failed to decode base64 content: %w", err)
		}
		f.Content = decoded
	case "plain":
		f.Content = []byte(aux.Content)
	default:
		return fmt.Errorf("unsupported encoding: %s", aux.Encoding)
	}

	return nil
}

type GroupData struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Data        map[string]string `json:"meta-data,omitempty"`
	File        CloudConfigFile   `json:"file,omitempty"`
}

type MetaData struct {
	InstanceID    string       `json:"instance-id" yaml:"instance-id"`
	LocalHostname string       `json:"local-hostname" yaml:"local-hostname"`
	Hostname      string       `json:"hostname" yaml:"hostname"`
	ClusterName   string       `json:"cluster-name" yaml:"cluster-name"`
	InstanceData  InstanceData `json:"instance-data" yaml:"instance-data"`
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

type OpenCHAMIComponent struct {
	base.Component
	MAC string `json:"mac"`
	IP  string `json:"ip"`
}
