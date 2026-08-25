//go:build stress

package wgtunnel

import (
	"fmt"
	"sync"
	"testing"
)

func TestStressIpForPeerConcurrent10K(t *testing.T) {
	manager := newTestInterfaceManager(t)

	const peerCount = 10_000
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

			peerName := fmt.Sprintf("10.%d.%d.%d", i/65536, (i/256)%256, i%256)
			allocated <- manager.IpForPeer(peerName, fmt.Sprintf("key-%d", i))
		}()
	}

	ready.Wait()
	start.Done()
	done.Wait()
	close(allocated)

	seen := make(map[string]struct{}, peerCount)
	for ip := range allocated {
		if ip == "" {
			t.Fatal("expected allocated IP, got empty string")
		}
		if _, found := seen[ip]; found {
			t.Fatalf("IP %s allocated more than once", ip)
		}
		seen[ip] = struct{}{}
	}
	if len(seen) != peerCount {
		t.Fatalf("expected %d allocations, got %d", peerCount, len(seen))
	}
}
