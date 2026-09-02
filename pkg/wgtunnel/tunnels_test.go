// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
// SPDX-License-Identifier: MIT

package wgtunnel

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestInterfaceManager(t *testing.T) *InterfaceManager {
	t.Helper()

	_, network, err := net.ParseCIDR("10.89.0.0/16")
	if err != nil {
		t.Fatalf("failed to parse test network: %v", err)
	}

	allocator, err := NewIPAllocator(network.String())
	if err != nil {
		t.Fatalf("failed to create allocator: %v", err)
	}

	serverIP := net.IPAddr{IP: net.ParseIP("10.89.0.1")}
	if err := allocator.Reserve(serverIP); err != nil {
		t.Fatalf("failed to reserve server IP: %v", err)
	}

	return &InterfaceManager{
		interfaceName: "wg0",
		network:       *network,
		ipAddress:     serverIP,
		peers:         make(map[string]PeerConfig),
		peersMutex:    sync.RWMutex{},
		ipManager:     allocator,
	}
}

func TestIpForPeerConcurrentAllocations(t *testing.T) {
	manager := newTestInterfaceManager(t)

	const peerCount = 512
	var ready sync.WaitGroup
	var start sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(peerCount)
	start.Add(1)
	done.Add(peerCount)

	allocated := make(chan string, peerCount)
	for i := range peerCount {
		go func() {
			defer done.Done()
			ready.Done()
			start.Wait()

			peerName := fmt.Sprintf("10.1.%d.%d", i/256, i%256)
			allocated <- manager.IpForPeer(peerName, fmt.Sprintf("key-%d", i))
		}()
	}

	ready.Wait()
	start.Done()
	done.Wait()
	close(allocated)

	seen := make(map[string]bool, peerCount)
	for ip := range allocated {
		if ip == "" {
			t.Fatal("expected allocated IP, got empty string")
		}
		if seen[ip] {
			t.Fatalf("IP %s allocated more than once", ip)
		}
		seen[ip] = true
	}
	if len(seen) != peerCount {
		t.Fatalf("expected %d allocations, got %d", peerCount, len(seen))
	}
}

func TestGetPeersReturnsCopy(t *testing.T) {
	manager := newTestInterfaceManager(t)
	peerIP := manager.IpForPeer("10.1.0.1", "key-1")
	if peerIP == "" {
		t.Fatal("expected allocated peer IP")
	}

	peers := manager.GetPeers()
	peers["10.1.0.2"] = PeerConfig{PublicKey: "key-2"}

	manager.peersMutex.RLock()
	defer manager.peersMutex.RUnlock()
	if _, found := manager.peers["10.1.0.2"]; found {
		t.Fatal("GetPeers returned mutable internal peers map")
	}
}
func TestAddPeerConfiguresWireGuardWithoutMutatingPeers(t *testing.T) {
	manager := newTestInterfaceManager(t)
	clientIP := "10.1.0.1"
	publicKey := "key-1"
	vpnIP := manager.IpForPeer(clientIP, publicKey)
	if vpnIP == "" {
		t.Fatal("expected allocated peer IP")
	}

	argsFile := installFakeWG(t)
	if err := manager.AddPeer(publicKey, vpnIP, clientIP); err != nil {
		t.Fatalf("AddPeer() error = %v, want nil", err)
	}

	manager.peersMutex.RLock()
	defer manager.peersMutex.RUnlock()
	if _, found := manager.peers[manager.GetInterfaceName()]; found {
		t.Fatalf("AddPeer wrote peer under interface name %q", manager.GetInterfaceName())
	}
	peer, found := manager.peers[clientIP]
	if !found {
		t.Fatalf("peer %q missing after IpForPeer", clientIP)
	}
	if peer.PublicKey != publicKey || peer.IP.IP.String() != vpnIP {
		t.Fatalf("peer = %+v, want public key %q and IP %q", peer, publicKey, vpnIP)
	}

	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("failed to read fake wg args: %v", err)
	}
	got := strings.TrimSpace(string(args))
	want := fmt.Sprintf("set wg0 peer %s allowed-ips %s/32", publicKey, vpnIP)
	if got != want {
		t.Fatalf("wg args = %q, want %q", got, want)
	}
}

func TestRemovePeerRunsWireGuardOutsidePeerLock(t *testing.T) {
	manager := newTestInterfaceManager(t)
	peerName := "10.1.0.1"
	originalKey := "key-1"
	vpnIP := manager.IpForPeer(peerName, originalKey)
	if vpnIP == "" {
		t.Fatal("expected allocated peer IP")
	}

	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseWG := func() {
		releaseOnce.Do(func() {
			close(release)
		})
	}
	defer releaseWG()
	argsFile := installBlockingFakeWG(t, release)
	removeDone := make(chan error, 1)
	go func() {
		removeDone <- manager.RemovePeer(peerName)
	}()

	waitForWGArgs(t, argsFile)
	if got := manager.IpForPeer(peerName, "key-2"); got != vpnIP {
		t.Fatalf("IpForPeer while remove was blocked = %q, want %q", got, vpnIP)
	}
	releaseWG()
	if err := <-removeDone; err != nil {
		t.Fatalf("RemovePeer() error = %v", err)
	}

	manager.peersMutex.RLock()
	defer manager.peersMutex.RUnlock()
	peer, found := manager.peers[peerName]
	if !found {
		t.Fatal("RemovePeer deleted peer that was replaced while wg command ran")
	}
	if peer.PublicKey != "key-2" {
		t.Fatalf("peer public key = %q, want replacement key", peer.PublicKey)
	}

	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("failed to read fake wg args: %v", err)
	}
	want := fmt.Sprintf("set wg0 peer %s remove", originalKey)
	if got := strings.TrimSpace(string(args)); got != want {
		t.Fatalf("wg args = %q, want %q", got, want)
	}
}

func TestRemovePeerReleasesAllocatedIP(t *testing.T) {
	manager := newTestInterfaceManager(t)
	peerName := "10.1.0.1"
	publicKey := "key-1"
	vpnIP := manager.IpForPeer(peerName, publicKey)
	if vpnIP == "" {
		t.Fatal("expected allocated peer IP")
	}

	installFakeWG(t)
	if err := manager.RemovePeer(peerName); err != nil {
		t.Fatalf("RemovePeer() error = %v, want nil", err)
	}

	reusedIP := manager.IpForPeer("10.1.0.2", "key-2")
	if reusedIP != vpnIP {
		t.Fatalf("reused IP = %q, want released IP %q", reusedIP, vpnIP)
	}
}

func installFakeWG(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	argsFile := filepath.Join(dir, "wg-args")
	wgPath := filepath.Join(dir, "wg")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\n", argsFile)
	if err := os.WriteFile(wgPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake wg: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return argsFile
}

func installBlockingFakeWG(t *testing.T, release <-chan struct{}) string {
	t.Helper()

	dir := t.TempDir()
	argsFile := filepath.Join(dir, "wg-args")
	releaseFile := filepath.Join(dir, "release")
	wgPath := filepath.Join(dir, "wg")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\nwhile [ ! -f %q ]; do sleep 0.01; done\n", argsFile, releaseFile)
	if err := os.WriteFile(wgPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake wg: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	go func() {
		<-release
		_ = os.WriteFile(releaseFile, []byte("release"), 0o644)
	}()
	return argsFile
}

func waitForWGArgs(t *testing.T, argsFile string) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("fake wg command did not start")
		default:
			if _, err := os.Stat(argsFile); err == nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}
