package smdclient

import (
	"net"
	"strings"
	"testing"
)

func TestIncrementXname(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		err      bool
	}{
		{"x0c0b0n0", "x0c0b0n1", false},
		{"x0c0b0n3", "x0c0b1n0", false},
		{"x0c0b7n3", "x0c1b0n0", false},
		{"x0c3b7n3", "x1c0b0n0", false},
		{"x0c0b0n4", "", true},
		{"x0c0b0", "", true},
		{"x0c0b", "", true},
		{"x0c0", "", true},
		{"x0", "", true},
		{"", "", true},
	}

	for _, test := range tests {
		result, err := incrementXname(test.input)
		if (err != nil) != test.err {
			t.Errorf("incrementXname(%s) error = %v, expected error = %v", test.input, err, test.err)
		}
		if result != test.expected {
			t.Errorf("incrementXname(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestGenerateFakeComponents(t *testing.T) {
	tests := []struct {
		numComponents int
		cidr          string
		expectError   bool
	}{
		{10, "10.20.30.0/24", false},
		{256, "10.20.30.0/24", true},
		{50, "invalidCIDR", true},
	}

	for _, test := range tests {
		components, rosettaMapping, err := generateFakeComponents(test.numComponents, test.cidr)
		if (err != nil) != test.expectError {
			t.Errorf("generateFakeComponents(%d, %s) error = %v, expected error = %v", test.numComponents, test.cidr, err, test.expectError)
		}
		if !test.expectError {
			if len(components) != test.numComponents {
				t.Errorf("expected %d components, got %d", test.numComponents, len(components))
			}
			if len(rosettaMapping) != test.numComponents {
				t.Errorf("expected %d rosetta mappings, got %d", test.numComponents, len(rosettaMapping))
			}
		}
	}
}

func TestIncrementIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"10.20.30.0", "10.20.30.1"},
		{"10.20.30.255", "10.20.31.0"},
		{"10.20.255.255", "10.21.0.0"},
	}

	for _, test := range tests {
		ip := net.ParseIP(test.input)
		result := incrementIP(ip).String()
		if result != test.expected {
			t.Errorf("incrementIP(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestIncrementMAC(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"00:DE:AD:BE:EF:00", "00:DE:AD:BE:EF:01"},
		{"00:DE:AD:BE:EF:FF", "00:DE:AD:BE:F0:00"},
	}

	for _, test := range tests {
		result := incrementMAC(test.input)
		if result != test.expected {
			t.Errorf("incrementMAC(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestNewFakeSMDClient(t *testing.T) {
	client := NewFakeSMDClient("fake", 50)

	if len(client.components) != 50 {
		t.Errorf("expected 50 components, got %d", len(client.components))
	}

	if len(client.rosetta_mapping) != 50 {
		t.Errorf("expected 50 rosetta mappings, got %d", len(client.rosetta_mapping))
	}

	// Check if groups are created correctly
	expectedCabinets := make(map[string]bool)
	for _, c := range client.rosetta_mapping {
		cabinet := strings.Split(c.ComponentID, "c")[0]
		expectedCabinets[cabinet] = true
	}

	for cabinet := range expectedCabinets {
		if _, ok := client.groups[cabinet]; !ok {
			t.Errorf("expected group for cabinet %s, but not found", cabinet)
		}
	}

	if len(client.groups["compute"]) != 46 {
		t.Errorf("expected 45 components in compute group, got %d", len(client.groups["compute"]))
	}

	if len(client.groups["io"]) != 9 {
		t.Errorf("expected 9 components in io group, got %d", len(client.groups["io"]))
	}
}
