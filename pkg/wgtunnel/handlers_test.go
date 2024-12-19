package wgtunnel

import (
	"net"
	"testing"
)

func TestNextIP(t *testing.T) {
	ip, network, _ := net.ParseCIDR("192.168.1.0/24")
	manager := NewInterfaceManager("wg0", ip, network)

	tests := []struct {
		name          string
		expectedIP    string
		lastAllocated *net.IPAddr
	}{
		{
			name:          "First IP allocation",
			expectedIP:    "192.168.1.0",
			lastAllocated: nil,
		},
		{
			name:          "Second IP allocation",
			expectedIP:    "192.168.1.1",
			lastAllocated: &net.IPAddr{IP: net.ParseIP("192.168.1.0")},
		},
		{
			name:          "Third IP allocation",
			expectedIP:    "192.168.1.2",
			lastAllocated: &net.IPAddr{IP: net.ParseIP("192.168.1.1")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.lastallocatedIP = tt.lastAllocated
			ip := manager.nextIP()
			if ip.IP.String() != tt.expectedIP {
				t.Errorf("expected %s, got %s", tt.expectedIP, ip.IP.String())
			}
		})
	}
}
