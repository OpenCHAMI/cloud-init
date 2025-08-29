package wgtunnel

import (
	"net"
	"strconv"
	"testing"
)

func TestNewIPAllocator(t *testing.T) {
	_, err := NewIPAllocator("192.168.1.0/24")
	if err != nil {
		t.Fatalf("Failed to create IPAllocator: %v", err)
	}
}

func TestReserve(t *testing.T) {
	allocator, _ := NewIPAllocator("192.168.1.0/24")
	ip := net.IPAddr{IP: net.ParseIP("192.168.1.10")}
	err := allocator.Reserve(ip)
	if err != nil {
		t.Fatalf("Failed to reserve IP: %v", err)
	}

	// Try to reserve the same IP again
	err = allocator.Reserve(ip)
	if err == nil {
		t.Fatalf("Expected error when reserving the same IP twice")
	}
}

func TestNextAvailable(t *testing.T) {
	allocator, _ := NewIPAllocator("192.168.1.0/24")

	// Reserve the first few IPs to test allocation
	for i := 0; i <= 253; i++ {
		ip, err := allocator.NextAvailable()
		if err != nil {
			t.Fatalf("Failed to get next available IP: %v", err)
		}
		expectedIP := net.IPAddr{IP: net.ParseIP("192.168.1." + strconv.Itoa(i+1))}
		if !ip.IP.Equal(expectedIP.IP) {
			t.Fatalf("Expected IP %v, got %v", expectedIP, ip)
		}
	}

	// Ensure network and broadcast addresses are not allocated
	networkIP := net.IPAddr{IP: net.ParseIP("192.168.1.0")}
	broadcastIP := net.IPAddr{IP: net.ParseIP("192.168.1.255")}
	if allocator.IsAllocated(networkIP) {
		t.Fatalf("Network address should not be allocated")
	}
	if allocator.IsAllocated(broadcastIP) {
		t.Fatalf("Broadcast address should not be allocated")
	}
}

func TestIsAllocated(t *testing.T) {
	allocator, _ := NewIPAllocator("192.168.1.0/24")
	ip := net.IPAddr{IP: net.ParseIP("192.168.1.10")}
	_ = allocator.Reserve(ip)

	if !allocator.IsAllocated(ip) {
		t.Fatalf("Expected IP to be allocated")
	}

	nonAllocatedIP := net.IPAddr{IP: net.ParseIP("192.168.1.20")}
	if allocator.IsAllocated(nonAllocatedIP) {
		t.Fatalf("Expected IP to not be allocated")
	}
}

func TestRelease(t *testing.T) {
	allocator, _ := NewIPAllocator("192.168.1.0/24")
	ip := net.IPAddr{IP: net.ParseIP("192.168.1.10")}
	_ = allocator.Reserve(ip)

	err := allocator.Release(ip)
	if err != nil {
		t.Fatalf("Failed to release IP: %v", err)
	}

	if allocator.IsAllocated(ip) {
		t.Fatalf("Expected IP to be released")
	}

	// Try to release an IP that is not allocated
	err = allocator.Release(ip)
	if err == nil {
		t.Fatalf("Expected error when releasing a non-allocated IP")
	}
}
