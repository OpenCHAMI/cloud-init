package cistore

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	base "github.com/Cray-HPE/hms-base"
)

type GroupData struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Data        map[string]string `json:"meta-data,omitempty"`
	File        CloudConfigFile   `json:"file,omitempty"`
}

type OpenCHAMIComponent struct {
	base.Component
	MAC string `json:"mac"`
	IP  string `json:"ip"`
}

type OpenCHAMIInstanceInfo struct {
	ID               string   `json:"id"`
	InstanceID       string   `json:"instance-id" yaml:"instance-id"`
	LocalHostname    string   `json:"local-hostname,omitempty" yaml:"local-hostname"`
	Hostname         string   `json:"hostname,omitempty" yaml:"hostname"`
	ClusterName      string   `json:"cluster-name,omitempty" yaml:"cluster-name"`
	Region           string   `json:"region,omitempty" yaml:"region"`
	AvailabilityZone string   `json:"availability-zone,omitempty" yaml:"availability-zone"`
	CloudProvider    string   `json:"cloud-provider,omitempty" yaml:"cloud-provider"`
	InstanceType     string   `json:"instance-type,omitempty" yaml:"instance-type"`
	CloudInitBaseURL string   `json:"cloud-init-base-url,omitempty" yaml:"cloud-init-base-url"`
	PublicKeys       []string `json:"public-keys,omitempty" yaml:"public-keys,omitempty"`
}

type ClusterDefaults struct {
	CloudProvider    string   `json:"cloud_provider,omitempty" yaml:"cloud-provider,omitempty"`
	Region           string   `json:"region,omitempty" yaml:"region,omitempty"`
	AvailabilityZone string   `json:"availability-zone,omitempty" yaml:"availability-zone,omitempty"`
	ClusterName      string   `json:"cluster-name,omitempty" yaml:"cluster-name,omitempty"`
	PublicKeys       []string `json:"public-keys,omitempty" yaml:"public-keys,omitempty"`
	BaseUrl          string   `json:"base-url,omitempty" yaml:"base-url,omitempty"`
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
