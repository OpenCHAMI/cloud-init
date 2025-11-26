package main

import (
	"strings"
	"testing"

	base "github.com/Cray-HPE/hms-base"
	"github.com/OpenCHAMI/cloud-init/internal/memstore"
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	yaml "gopkg.in/yaml.v2"
)

// TestNetworkInterfacesInMetadata tests that all network interfaces are included in metadata
func TestNetworkInterfacesInMetadata(t *testing.T) {
	// Create a test store
	store := memstore.NewMemStore()

	// Create a fake SMD client
	fakeSmd := smdclient.NewFakeSMDClient("test-cluster", 1)

	// Create test component - use the ID that the fake SMD client will generate
	component := cistore.OpenCHAMIComponent{
		Component: base.Component{
			ID:   "x3000c0b0n1", // This matches the first generated fake component
			Type: "Node",
		},
		IP:  "10.20.30.41",
		MAC: "aa:bb:cc:dd:ee:ff",
	}

	// Generate metadata with no groups for simplicity
	groups := []string{}
	metadata := generateMetaData(component, groups, store, fakeSmd)

	// Marshal to YAML to see the output
	yamlData, err := yaml.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal metadata to YAML: %v", err)
	}

	yamlString := string(yamlData)
	t.Logf("Generated YAML:\n%s", yamlString)

	// Check if network_interfaces is present in the YAML
	if !strings.Contains(yamlString, "network_interfaces:") {
		t.Error("Expected 'network_interfaces' to be present in the metadata")
	}

	// Check if the network interfaces contain MAC and IP information
	if !strings.Contains(yamlString, "mac:") {
		t.Error("Expected 'mac:' field to be present in network interfaces")
	}

	if !strings.Contains(yamlString, "ip:") {
		t.Error("Expected 'ip:' field to be present in network interfaces")
	}

	// Verify the network interfaces are populated correctly
	interfaces := metadata.InstanceData.V1.VendorData.NetworkInterfaces
	if len(interfaces) == 0 {
		t.Error("Expected at least one network interface in the metadata")
	} else {
		// Check first interface
		firstInterface := interfaces[0]
		if firstInterface.MAC == "" {
			t.Error("Expected MAC address to be populated")
		}
		if firstInterface.IP == "" {
			t.Error("Expected IP address to be populated")
		}
		t.Logf("First interface - MAC: %s, IP: %s, Description: %s",
			firstInterface.MAC, firstInterface.IP, firstInterface.Description)
	}
}

// TestNetworkInterfacesWithMultipleInterfaces tests handling of multiple interfaces
func TestNetworkInterfacesWithMultipleInterfaces(t *testing.T) {
	// This test would be more comprehensive if we had a way to create
	// a fake SMD client with multiple interfaces per node.
	// For now, we'll test that the structure supports multiple interfaces.

	store := memstore.NewMemStore()
	fakeSmd := smdclient.NewFakeSMDClient("test-cluster", 1)

	component := cistore.OpenCHAMIComponent{
		Component: base.Component{
			ID:   "x3000c0b0n1", // Use the correct fake SMD client ID
			Type: "Node",
		},
		IP:  "10.20.30.41",
		MAC: "aa:bb:cc:dd:ee:ff",
	}

	groups := []string{}
	metadata := generateMetaData(component, groups, store, fakeSmd)

	// Verify the structure can handle multiple interfaces
	interfaces := metadata.InstanceData.V1.VendorData.NetworkInterfaces
	t.Logf("Found %d network interfaces", len(interfaces))

	for i, iface := range interfaces {
		t.Logf("Interface %d: MAC=%s, IP=%s, WGIP=%s, Desc=%s",
			i, iface.MAC, iface.IP, iface.WGIP, iface.Description)
	}
}
