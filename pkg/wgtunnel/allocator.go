package wgtunnel

import (
	"errors"
	"net"
	"sync"
)

// IPAllocator manages IP address allocation within a network range.
type IPAllocator struct {
	network       *net.IPNet
	usedIPs       map[string]bool
	mu            sync.Mutex
	networkAddr   net.IP
	broadcastAddr net.IP
}

// NewIPAllocator initializes a new IPAllocator for a given network.
func NewIPAllocator(cidr string) (*IPAllocator, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	ip := network.IP.To4()
	if ip == nil {
		return nil, errors.New("only IPv4 is supported")
	}

	// Calculate the network and broadcast addresses
	networkAddr := network.IP.Mask(network.Mask)
	broadcastAddr := make(net.IP, len(networkAddr))
	for i := range networkAddr {
		broadcastAddr[i] = networkAddr[i] | ^network.Mask[i]
	}

	return &IPAllocator{
		network:       network,
		networkAddr:   networkAddr,
		broadcastAddr: broadcastAddr,
		usedIPs:       make(map[string]bool),
	}, nil
}

// Reserve reserves a specific IP address.
func (a *IPAllocator) Reserve(ipAddr net.IPAddr) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	ip := ipAddr.IP
	if !a.network.Contains(ip) {
		return errors.New("IP address out of range")
	}
	ipStr := ip.String()
	if a.usedIPs[ipStr] {
		return errors.New("IP address already allocated")
	}
	a.usedIPs[ipStr] = true
	return nil
}

// NextAvailable returns the next available IP address in the range.
func (a *IPAllocator) NextAvailable() (net.IPAddr, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ip := make(net.IP, len(a.networkAddr))
	copy(ip, a.networkAddr)
	for {
		for i := len(ip) - 1; i >= 0; i-- {
			ip[i]++
			if ip[i] != 0 {
				break
			}
		}

		// Check if the incremented IP is still within the subnet range
		if !a.network.Contains(ip) {
			return net.IPAddr{}, errors.New("IP range exhausted: no available IP addresses in range " + a.network.String())
		}

		ipStr := ip.String()
		if !a.usedIPs[ipStr] {
			a.usedIPs[ipStr] = true
			return net.IPAddr{IP: ip}, nil
		}
	}
}

// IsAllocated checks if an IP address is currently allocated.
func (a *IPAllocator) IsAllocated(ipAddr net.IPAddr) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.usedIPs[ipAddr.IP.String()]
}

// Release releases an IP address back to the pool.
func (a *IPAllocator) Release(ipAddr net.IPAddr) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	ipStr := ipAddr.IP.String()
	if !a.usedIPs[ipStr] {
		return errors.New("IP address not allocated")
	}
	delete(a.usedIPs, ipStr)
	return nil
}
