package bss

import (
	"encoding/json"
	"errors"
	"strings"
)

// BootMethod defines the supported boot methods
type BootMethod string

// RootFS defines the root filesystem configuration
type RootFS struct {
	Type    string `json:"type,omitempty"`    // Type of root filesystem (nfs, local, etc.)
	Server  string `json:"server,omitempty"`  // Server for network filesystems
	Path    string `json:"path,omitempty"`    // Path to root filesystem
	Options string `json:"options,omitempty"` // Mount options
}

// CloudInitServer defines the cloud-init server configuration
type CloudInitServer struct {
	URL     string `json:"url,omitempty"`     // URL of the cloud-init server
	Version string `json:"version,omitempty"` // Version of the cloud-init server
}

// BootParams (V2) defines the boot parameters for one or more nodes
type BootParams struct {
	Version   int              `json:"version,omitempty"`    // Version of the boot parameters
	Params    string           `json:"params,omitempty"`     // Kernel boot parameters
	Kernel    string           `json:"kernel,omitempty"`     // Kernel image URL/path
	Initrd    string           `json:"initrd,omitempty"`     // Initrd image URL/path
	RootFS    *RootFS          `json:"rootfs,omitempty"`     // Root filesystem configuration
	CloudInit *CloudInitServer `json:"cloud-init,omitempty"` // Cloud-init server configuration
}

// BootParamsV1 defines the boot parameters for one or more nodes
// This is the old version that matches the BSS API specification
type BootParamsV1 struct {
	Hosts         []string         `json:"hosts,omitempty"`          // List of host xnames
	MACs          []string         `json:"macs,omitempty"`           // List of MAC addresses
	NIDs          []int            `json:"nids,omitempty"`           // List of Node IDs
	Group         string           `json:"group,omitempty"`          // Group name
	Params        string           `json:"params,omitempty"`         // Kernel boot parameters
	Kernel        string           `json:"kernel,omitempty"`         // Kernel image URL/path
	Initrd        string           `json:"initrd,omitempty"`         // Initrd image URL/path
	CloudInit     *CloudInitServer `json:"cloud-init,omitempty"`     // Cloud-init server configuration
	ReferralToken string           `json:"referral_token,omitempty"` // Referral token
}

// ParseFromJSON unmarshals a JSON byte array into a BootParams struct
func (b *BootParams) ParseFromJSON(body []byte) error {
	if err := json.Unmarshal(body, b); err != nil {
		return err
	}

	// Validate that kernel and initrd are provided
	if b.Initrd == "" || b.Kernel == "" {
		return errors.New("kernel and initrd must both be specified")
	}

	return nil
}

// ParseFromJSON unmarshals a JSON byte array into a BootParamsV1 struct
func (b *BootParamsV1) ParseFromJSON(body []byte) error {
	if err := json.Unmarshal(body, b); err != nil {
		return err
	}

	// Validate that at least one of hosts, MACs, or NIDs is provided
	if len(b.Hosts) == 0 && len(b.MACs) == 0 && len(b.NIDs) == 0 {
		return errors.New("at least one of hosts, MACs, or NIDs must be specified")
	}

	// Validate that kernel and initrd are provided
	if b.Initrd == "" || b.Kernel == "" {
		return errors.New("kernel and initrd must both be specified")
	}

	return nil
}

// ValidateBootParams validates the boot parameters
func (b *BootParams) ValidateBootParams() error {
	if b.Kernel == "" || b.Initrd == "" {
		return errors.New("kernel and initrd must both be specified")
	}
	return nil
}

// MergeBootParams merges multiple boot parameters into a single boot parameter
// It will take the first kernel and initrd that are not empty
// It will also take the first rootfs and cloud-init that are not nil
func MergeBootParams(params []*BootParams) (*BootParams, error) {
	merged := &BootParams{}
	for _, param := range params {
		merged.Params += param.Params + " "
		// If the kernel is not set, use the first one that also has an initrd
		if merged.Kernel == "" && param.Kernel != "" && param.Initrd != "" {
			merged.Kernel = param.Kernel
			merged.Initrd = param.Initrd
		}
		if merged.RootFS == nil && param.RootFS != nil {
			merged.RootFS = param.RootFS
		}
		if merged.CloudInit == nil && param.CloudInit != nil {
			merged.CloudInit = param.CloudInit
		}
	}
	merged.Params = strings.TrimSpace(merged.Params)
	return merged, nil
}
