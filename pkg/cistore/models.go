package cistore

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	base "github.com/Cray-HPE/hms-base"
)

type GroupData struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Data        map[string]interface{} `json:"meta-data,omitempty"`
	File        CloudConfigFile        `json:"file,omitempty"`
}

func (g *GroupData) ParseFromJSON(body []byte) error {
	// Parse the JSON
	if err := json.Unmarshal(body, g); err != nil {
		return err
	}

	// Perform validation
	if g.Name == "" {
		return errors.New("name is required")
	}

	return nil
}

type OpenCHAMIComponent struct {
	base.Component
	MAC  string `json:"mac"`            // MAC address of the inteface used to boot the component
	IP   string `json:"ip"`             // IP address of the interface used to boot the component
	WGIP string `json:"wgip,omitempty"` // Wireguard IP address of the interface used for cloud-init
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
	BootSubnet       string   `json:"boot-subnet,omitempty" yaml:"boot-subnet,omitempty"`
	WGSubnet         string   `json:"wg-subnet,omitempty" yaml:"wg-subnet,omitempty"`
	ShortName        string   `json:"short-name,omitempty" yaml:"short-name,omitempty"`
	NidLength        int      `json:"nid-length,omitempty" yaml:"nid-length,omitempty"`
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
