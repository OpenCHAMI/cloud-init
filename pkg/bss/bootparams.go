package bss

import (
	"encoding/json"
	"errors"
)

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

// BootParams defines the boot parameters for one or more nodes
type BootParams struct {
	Version   int              `json:"version,omitempty"`    // Version of the boot parameters
	Params    string           `json:"params,omitempty"`     // Kernel boot parameters
	Kernel    string           `json:"kernel,omitempty"`     // Kernel image URL/path
	Initrd    string           `json:"initrd,omitempty"`     // Initrd image URL/path
	RootFS    *RootFS          `json:"rootfs,omitempty"`     // Root filesystem configuration
	CloudInit *CloudInitServer `json:"cloud-init,omitempty"` // Cloud-init server configuration
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
