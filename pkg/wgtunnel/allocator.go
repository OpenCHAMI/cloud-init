// SPDX-FileCopyrightText: 2026 Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

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
	nextIP        net.IP
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
		nextIP:        nextIP(networkAddr),
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
	if ip.Equal(a.nextIP) {
		a.nextIP = nextIP(ip)
	}
	return nil
}

// NextAvailable returns the next available IP address in the range.
func (a *IPAllocator) NextAvailable() (net.IPAddr, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ip := cloneIP(a.nextIP)
	start := cloneIP(a.nextIP)
	wrapped := false
	for {
		if wrapped && ip.Equal(start) {
			return net.IPAddr{}, errors.New("IP range exhausted: no available IP addresses in range " + a.network.String())
		}

		if !a.network.Contains(ip) || ip.Equal(a.networkAddr) || ip.Equal(a.broadcastAddr) {
			ip = a.firstUsableIP()
			wrapped = true
			continue
		}

		ipStr := ip.String()
		if !a.usedIPs[ipStr] {
			a.usedIPs[ipStr] = true
			a.nextIP = nextIP(ip)
			return net.IPAddr{IP: ip}, nil
		}
		ip = nextIP(ip)
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
	if a.network.Contains(ipAddr.IP) && !ipAddr.IP.Equal(a.networkAddr) && !ipAddr.IP.Equal(a.broadcastAddr) && ipLess(ipAddr.IP, a.nextIP) {
		a.nextIP = cloneIP(ipAddr.IP)
	}
	return nil
}

func (a *IPAllocator) firstUsableIP() net.IP {
	return nextIP(a.networkAddr)
}

func cloneIP(ip net.IP) net.IP {
	clone := make(net.IP, len(ip))
	copy(clone, ip)
	return clone
}

func nextIP(ip net.IP) net.IP {
	next := cloneIP(ip)
	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}
	return next
}

func ipLess(left, right net.IP) bool {
	left = left.To4()
	right = right.To4()
	if left == nil || right == nil {
		return false
	}
	for i := range left {
		if left[i] < right[i] {
			return true
		}
		if left[i] > right[i] {
			return false
		}
	}
	return false
}
