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

type OpenCHAMIComponent struct {
	base.Component
	MAC string `json:"mac"`
	IP  string `json:"ip"`
}

type OpenCHAMIInstanceInfo struct {
	ID               string `json:"id"`
	InstanceID       string `json:"instance-id" yaml:"instance-id"`
	LocalHostname    string `json:"local-hostname,omitempty" yaml:"local-hostname"`
	Hostname         string `json:"hostname,omitempty" yaml:"hostname"`
	ClusterName      string `json:"cluster-name,omitempty" yaml:"cluster-name"`
	Region           string `json:"region,omitempty" yaml:"region"`
	AvailabilityZone string `json:"availability-zone,omitempty" yaml:"availability-zone"`
	CloudProvider    string `json:"cloud-provider,omitempty" yaml:"cloud-provider"`
}
