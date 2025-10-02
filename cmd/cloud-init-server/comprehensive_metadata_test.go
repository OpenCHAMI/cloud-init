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

// TestCompleteMetadataWithNetworkInterfaces tests the complete metadata output including network interfaces
func TestCompleteMetadataWithNetworkInterfaces(t *testing.T) {
	// Create a test store with some group data
	store := memstore.NewMemStore()

	// Add cluster defaults
	clusterDefaults := cistore.ClusterDefaults{
		ClusterName:   "demo-cluster",
		ShortName:     "demo",
		NidLength:     4,
		CloudProvider: "openchami",
		Region:        "us-west-2",
	}
	err := store.SetClusterDefaults(clusterDefaults)
	if err != nil {
		t.Fatalf("Failed to set cluster defaults: %v", err)
	}

	// Add a group with metadata
	groupData := cistore.GroupData{
		Name:        "compute",
		Description: "Compute nodes",
		Data: map[string]interface{}{
			"syslog_server":      "192.168.1.10",
			"ntp_servers":        "pool.ntp.org time.nist.gov",
			"environment":        "production cluster",
			"management_network": "10.1.0.0/16",
		},
	}
	err = store.AddGroupData("compute", groupData)
	if err != nil {
		t.Fatalf("Failed to add group data: %v", err)
	}

	// Create a fake SMD client
	fakeSmd := smdclient.NewFakeSMDClient("demo-cluster", 1)

	// Create test component - use the ID that matches the fake SMD client
	component := cistore.OpenCHAMIComponent{
		Component: base.Component{
			ID:   "x3000c0b0n1",
			Type: "Node",
		},
		IP:  "10.20.30.1",        // This should match what the fake SMD generates
		MAC: "00:DE:AD:BE:EF:01", // This should match what the fake SMD generates
	}

	// Generate metadata with the compute group
	groups := []string{"compute"}
	metadata := generateMetaData(component, groups, store, fakeSmd)

	// Marshal to YAML to see the complete output
	yamlData, err := yaml.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal metadata to YAML: %v", err)
	}

	yamlString := string(yamlData)
	t.Logf("Complete metadata YAML:\n%s", yamlString)

	// Verify all the expected components are present

	// Basic metadata
	if metadata.ClusterName != "demo-cluster" {
		t.Errorf("Expected cluster name 'demo-cluster', got '%s'", metadata.ClusterName)
	}

	// Network interfaces
	interfaces := metadata.InstanceData.V1.VendorData.NetworkInterfaces
	if len(interfaces) == 0 {
		t.Error("Expected at least one network interface")
	} else {
		firstInterface := interfaces[0]
		if firstInterface.MAC == "" {
			t.Error("Expected MAC address to be populated")
		}
		if firstInterface.IP == "" {
			t.Error("Expected IP address to be populated")
		}
		t.Logf("Network interface: MAC=%s, IP=%s, Desc=%s",
			firstInterface.MAC, firstInterface.IP, firstInterface.Description)
	}

	// Group metadata
	groups_data := metadata.InstanceData.V1.VendorData.Groups
	if len(groups_data) == 0 {
		t.Error("Expected group data to be present")
	} else {
		computeGroup, exists := groups_data["compute"]
		if !exists {
			t.Error("Expected 'compute' group to be present")
		} else {
			// Check that our group metadata values with spaces are preserved
			if syslogServer, ok := computeGroup["syslog_server"]; ok {
				if syslogServer != "192.168.1.10" {
					t.Errorf("Expected syslog_server '192.168.1.10', got '%v'", syslogServer)
				}
			} else {
				t.Error("Expected 'syslog_server' to be present in compute group")
			}

			if ntpServers, ok := computeGroup["ntp_servers"]; ok {
				ntpStr := ntpServers.(string)
				if !strings.Contains(ntpStr, "pool.ntp.org") || !strings.Contains(ntpStr, "time.nist.gov") {
					t.Errorf("Expected ntp_servers to contain both servers, got '%v'", ntpServers)
				}
				// Verify no unexpected newlines in the value with spaces
				if strings.Contains(ntpStr, "\n") {
					t.Errorf("NTP servers value contains unexpected newlines: '%v'", ntpServers)
				}
			} else {
				t.Error("Expected 'ntp_servers' to be present in compute group")
			}
		}
	}

	// Verify in YAML output that values with spaces are handled correctly
	if strings.Contains(yamlString, "pool.ntp.org\ntime.nist.gov") {
		t.Error("NTP servers value was incorrectly split across lines in YAML")
	}

	if strings.Contains(yamlString, "production\ncluster") {
		t.Error("Environment value was incorrectly split across lines in YAML")
	}

	// Verify the YAML contains the expected network interface fields
	if !strings.Contains(yamlString, "network_interfaces:") {
		t.Error("Expected 'network_interfaces:' to be present in YAML output")
	}
}

// TestNetworkInterfacesPreserveSpacesInValues specifically tests that spaces in metadata don't cause newlines
func TestNetworkInterfacesPreserveSpacesInValues(t *testing.T) {
	store := memstore.NewMemStore()

	// Add group data with various space scenarios
	groupData := cistore.GroupData{
		Name:        "test-spaces",
		Description: "Group to test space handling",
		Data: map[string]interface{}{
			"single_space":    "hello world",
			"multiple_spaces": "value with   multiple   spaces",
			"leading_space":   " leading space",
			"trailing_space":  "trailing space ",
			"description":     "This is a long description with many words and spaces",
			"command":         "systemctl restart some-service --with-options",
			"url_with_spaces": "https://example.com/path with spaces/file.txt",
		},
	}
	err := store.AddGroupData("test-spaces", groupData)
	if err != nil {
		t.Fatalf("Failed to add group data: %v", err)
	}

	fakeSmd := smdclient.NewFakeSMDClient("test-cluster", 1)
	component := cistore.OpenCHAMIComponent{
		Component: base.Component{
			ID:   "x3000c0b0n1",
			Type: "Node",
		},
		IP:  "10.20.30.1",
		MAC: "00:DE:AD:BE:EF:01",
	}

	groups := []string{"test-spaces"}
	metadata := generateMetaData(component, groups, store, fakeSmd)

	yamlData, err := yaml.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal metadata to YAML: %v", err)
	}

	yamlString := string(yamlData)
	t.Logf("YAML with spaces test:\n%s", yamlString)

	// Check that none of the values with spaces were split into multiple lines
	testCases := []struct {
		original string
		broken   string
		name     string
	}{
		{"hello world", "hello\nworld", "single space"},
		{"multiple   spaces", "multiple\n", "multiple spaces"},
		{"This is a long description", "This is a\nlong", "long description"},
		{"systemctl restart some-service", "systemctl restart\nsome-service", "command with spaces"},
		{"path with spaces", "path with\nspaces", "URL with spaces"},
	}

	for _, tc := range testCases {
		if strings.Contains(yamlString, tc.broken) {
			t.Errorf("%s was incorrectly split across lines in YAML", tc.name)
		}
		if !strings.Contains(yamlString, tc.original) {
			t.Errorf("Expected '%s' to be present as single value in YAML", tc.original)
		}
	}
}
