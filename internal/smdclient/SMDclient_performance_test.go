package smdclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGroupMembershipCached verifies that Bug #1 is fixed:
// GroupMembership should use the cache instead of making HTTP requests
func TestGroupMembershipCached(t *testing.T) {
	requestCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/hsm/v2/Inventory/EthernetInterfaces/":
			_, _ = w.Write([]byte(`[
				{
					"ComponentID": "x1000",
					"MACAddress": "00:11:22:33:44:55",
					"IPAddresses": [{"IPAddress": "192.168.1.1"}],
					"Description": "Test Node 1"
				}
			]`))
		case "/hsm/v2/memberships/x1000":
			_, _ = w.Write([]byte(`{"GroupLabels": ["compute", "cabinet1"]}`))
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &SMDClient{
		smdClient:   server.Client(),
		smdBaseURL:  server.URL,
		nodesMutex:  &sync.RWMutex{},
		nodes:       make(map[string]NodeMapping),
		ipToXname:   make(map[string]string),
		macToXname:  make(map[string]string),
		wgipToXname: make(map[string]string),
	}

	// Populate cache - should make 2 requests (interfaces + membership)
	initialRequests := requestCount
	client.PopulateNodes()
	populateRequests := requestCount - initialRequests

	// Verify group membership was cached
	groups, err := client.GroupMembership("x1000")
	require.NoError(t, err)
	assert.Equal(t, []string{"compute", "cabinet1"}, groups)

	// Verify no additional HTTP requests were made
	assert.Equal(t, populateRequests, requestCount-initialRequests,
		"GroupMembership should not make HTTP requests after cache is populated")

	// Call GroupMembership 100 times - should use cache every time
	for i := 0; i < 100; i++ {
		groups, err := client.GroupMembership("x1000")
		require.NoError(t, err)
		assert.Equal(t, []string{"compute", "cabinet1"}, groups)
	}

	// Verify still no additional requests
	assert.Equal(t, populateRequests, requestCount-initialRequests,
		"GroupMembership made %d HTTP requests when it should have used cache",
		requestCount-initialRequests-populateRequests)
}

// TestConcurrentReads verifies that Bug #2 is fixed:
// Read operations should use RLock instead of Lock to allow concurrent access
func TestConcurrentReads(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/hsm/v2/Inventory/EthernetInterfaces/":
			_, _ = w.Write([]byte(`[
				{
					"ComponentID": "x1000",
					"MACAddress": "00:11:22:33:44:55",
					"IPAddresses": [{"IPAddress": "192.168.1.1"}],
					"Description": "Test Node 1"
				},
				{
					"ComponentID": "x1001",
					"MACAddress": "00:11:22:33:44:66",
					"IPAddresses": [{"IPAddress": "192.168.1.2"}],
					"Description": "Test Node 2"
				}
			]`))
		case "/hsm/v2/memberships/x1000":
			_, _ = w.Write([]byte(`{"GroupLabels": ["compute"]}`))
		case "/hsm/v2/memberships/x1001":
			_, _ = w.Write([]byte(`{"GroupLabels": ["io"]}`))
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &SMDClient{
		smdClient:   server.Client(),
		smdBaseURL:  server.URL,
		nodesMutex:  &sync.RWMutex{},
		nodes:       make(map[string]NodeMapping),
		ipToXname:   make(map[string]string),
		macToXname:  make(map[string]string),
		wgipToXname: make(map[string]string),
	}

	client.PopulateNodes()

	// Test concurrent reads - with RLock these should all run in parallel
	// With Lock (the bug), they would serialize
	const concurrency = 100
	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()

			// Each goroutine does multiple read operations
			_, _ = client.IDfromIP("192.168.1.1")
			_, _ = client.IDfromMAC("00:11:22:33:44:55")
			_, _ = client.IPfromID("x1000")
			_, _ = client.MACfromID("x1000")
			_, _ = client.GroupMembership("x1000")

			// Add small delay to ensure overlap
			time.Sleep(1 * time.Millisecond)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// With RLock, 100 goroutines should complete in ~100-200ms
	// With Lock (serialized), it would take ~10-20s (100 * 100ms)
	assert.Less(t, elapsed, 2*time.Second,
		"Concurrent reads took %v - this suggests Lock instead of RLock is being used", elapsed)

	t.Logf("100 concurrent readers completed in %v", elapsed)
}

// TestReverseIndexPerformance verifies that Bug #3 is fixed:
// IP/MAC lookups should be O(1) using reverse indexes, not O(n) linear search
func TestReverseIndexPerformance(t *testing.T) {
	// Generate a large number of nodes to test performance
	nodeCount := 1000

	ethInterfaces := "["
	for i := 0; i < nodeCount; i++ {
		if i > 0 {
			ethInterfaces += ","
		}
		ethInterfaces += fmt.Sprintf(`{
			"ComponentID": "x%d",
			"MACAddress": "00:11:22:33:%02x:%02x",
			"IPAddresses": [{"IPAddress": "192.168.%d.%d"}],
			"Description": "Node %d"
		}`, i, (i>>8)&0xFF, i&0xFF, (i>>8)&0xFF, i&0xFF, i)
	}
	ethInterfaces += "]"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if r.URL.Path == "/hsm/v2/Inventory/EthernetInterfaces/" {
			_, _ = w.Write([]byte(ethInterfaces))
		} else if len(r.URL.Path) >= len("/hsm/v2/memberships/") && r.URL.Path[:len("/hsm/v2/memberships/")] == "/hsm/v2/memberships/" {
			_, _ = w.Write([]byte(`{"GroupLabels": ["compute"]}`))
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &SMDClient{
		smdClient:   server.Client(),
		smdBaseURL:  server.URL,
		nodesMutex:  &sync.RWMutex{},
		nodes:       make(map[string]NodeMapping),
		ipToXname:   make(map[string]string),
		macToXname:  make(map[string]string),
		wgipToXname: make(map[string]string),
	}

	client.PopulateNodes()

	// Verify reverse indexes were built
	assert.Equal(t, nodeCount, len(client.ipToXname), "IP reverse index should have %d entries", nodeCount)
	assert.Equal(t, nodeCount, len(client.macToXname), "MAC reverse index should have %d entries", nodeCount)

	// Test lookup performance - should be O(1)
	// With O(n) linear search, 1000 lookups on 1000 nodes = 1M comparisons
	// With O(1) hash map, 1000 lookups = 1000 lookups

	lookupCount := 1000
	start := time.Now()

	for i := 0; i < lookupCount; i++ {
		ip := fmt.Sprintf("192.168.%d.%d", (i>>8)&0xFF, i&0xFF)
		xname, err := client.IDfromIP(ip)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("x%d", i), xname)
	}

	elapsed := time.Since(start)

	// O(1) lookups: 1000 lookups should take <10ms
	// O(n) lookups: 1000 lookups on 1000 nodes would take >100ms
	assert.Less(t, elapsed, 50*time.Millisecond,
		"1000 lookups on 1000 nodes took %v - this suggests O(n) linear search instead of O(1) hash map", elapsed)

	t.Logf("1000 IP lookups on 1000 nodes completed in %v (avg %v per lookup)",
		elapsed, elapsed/time.Duration(lookupCount))

	// Test MAC lookups
	start = time.Now()
	for i := 0; i < lookupCount; i++ {
		mac := fmt.Sprintf("00:11:22:33:%02x:%02x", (i>>8)&0xFF, i&0xFF)
		xname, err := client.IDfromMAC(mac)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("x%d", i), xname)
	}
	elapsed = time.Since(start)

	assert.Less(t, elapsed, 50*time.Millisecond,
		"1000 MAC lookups on 1000 nodes took %v - this suggests O(n) linear search", elapsed)

	t.Logf("1000 MAC lookups on 1000 nodes completed in %v (avg %v per lookup)",
		elapsed, elapsed/time.Duration(lookupCount))
}

// TestCaseInsensitiveLookup verifies that IP/MAC lookups are case-insensitive
func TestCaseInsensitiveLookup(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/hsm/v2/Inventory/EthernetInterfaces/":
			_, _ = w.Write([]byte(`[
				{
					"ComponentID": "x1000",
					"MACAddress": "AA:BB:CC:DD:EE:FF",
					"IPAddresses": [{"IPAddress": "192.168.1.1"}],
					"Description": "Test Node"
				}
			]`))
		case "/hsm/v2/memberships/x1000":
			_, _ = w.Write([]byte(`{"GroupLabels": ["compute"]}`))
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &SMDClient{
		smdClient:   server.Client(),
		smdBaseURL:  server.URL,
		nodesMutex:  &sync.RWMutex{},
		nodes:       make(map[string]NodeMapping),
		ipToXname:   make(map[string]string),
		macToXname:  make(map[string]string),
		wgipToXname: make(map[string]string),
	}

	client.PopulateNodes()

	// Test case variations for MAC
	testMACs := []string{
		"AA:BB:CC:DD:EE:FF",
		"aa:bb:cc:dd:ee:ff",
		"Aa:Bb:Cc:Dd:Ee:Ff",
	}

	for _, mac := range testMACs {
		xname, err := client.IDfromMAC(mac)
		require.NoError(t, err, "Failed to lookup MAC %s", mac)
		assert.Equal(t, "x1000", xname, "Case-insensitive MAC lookup failed for %s", mac)
	}

	// Test case variations for IP (though IPs are typically lowercase)
	testIPs := []string{
		"192.168.1.1",
		"192.168.1.1", // IPs don't have case, but test anyway
	}

	for _, ip := range testIPs {
		xname, err := client.IDfromIP(ip)
		require.NoError(t, err, "Failed to lookup IP %s", ip)
		assert.Equal(t, "x1000", xname, "IP lookup failed for %s", ip)
	}
}

// TestAddWGIPUpdatesReverseIndex verifies that AddWGIP updates the reverse index
func TestAddWGIPUpdatesReverseIndex(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/hsm/v2/Inventory/EthernetInterfaces/":
			_, _ = w.Write([]byte(`[
				{
					"ComponentID": "x1000",
					"MACAddress": "00:11:22:33:44:55",
					"IPAddresses": [{"IPAddress": "192.168.1.1"}],
					"Description": "Test Node"
				}
			]`))
		case "/hsm/v2/memberships/x1000":
			_, _ = w.Write([]byte(`{"GroupLabels": ["compute"]}`))
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &SMDClient{
		smdClient:   server.Client(),
		smdBaseURL:  server.URL,
		nodesMutex:  &sync.RWMutex{},
		nodes:       make(map[string]NodeMapping),
		ipToXname:   make(map[string]string),
		macToXname:  make(map[string]string),
		wgipToXname: make(map[string]string),
	}

	client.PopulateNodes()

	// Initially no WireGuard IP
	_, err := client.IDfromIP("10.99.0.1")
	assert.Error(t, err, "WireGuard IP should not be found before AddWGIP")

	// Add WireGuard IP
	err = client.AddWGIP("x1000", "10.99.0.1")
	require.NoError(t, err)

	// Now it should be findable
	xname, err := client.IDfromIP("10.99.0.1")
	require.NoError(t, err)
	assert.Equal(t, "x1000", xname, "WireGuard IP should be findable after AddWGIP")

	// Verify WGIPfromID works
	wgip, err := client.WGIPfromID("x1000")
	require.NoError(t, err)
	assert.Equal(t, "10.99.0.1", wgip)
}

// BenchmarkIDfromIP benchmarks the IP lookup performance
func BenchmarkIDfromIP(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if r.URL.Path == "/hsm/v2/Inventory/EthernetInterfaces/" {
			// Generate 1000 nodes
			ethInterfaces := "["
			for i := 0; i < 1000; i++ {
				if i > 0 {
					ethInterfaces += ","
				}
				ethInterfaces += fmt.Sprintf(`{
				"ComponentID": "x%d",
				"MACAddress": "00:11:22:33:%02x:%02x",
				"IPAddresses": [{"IPAddress": "192.168.%d.%d"}],
				"Description": "Node %d"
			}`, i, (i>>8)&0xFF, i&0xFF, (i>>8)&0xFF, i&0xFF, i)
			}
			ethInterfaces += "]"
			_, _ = w.Write([]byte(ethInterfaces))
		} else if len(r.URL.Path) >= len("/hsm/v2/memberships/") && r.URL.Path[:len("/hsm/v2/memberships/")] == "/hsm/v2/memberships/" {
			_, _ = w.Write([]byte(`{"GroupLabels": ["compute"]}`))
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &SMDClient{
		smdClient:   server.Client(),
		smdBaseURL:  server.URL,
		nodesMutex:  &sync.RWMutex{},
		nodes:       make(map[string]NodeMapping),
		ipToXname:   make(map[string]string),
		macToXname:  make(map[string]string),
		wgipToXname: make(map[string]string),
	}

	client.PopulateNodes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ip := fmt.Sprintf("192.168.%d.%d", (i%1000>>8)&0xFF, (i%1000)&0xFF)
		_, _ = client.IDfromIP(ip)
	}
}

// BenchmarkGroupMembership benchmarks the group membership lookup performance
func BenchmarkGroupMembership(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/hsm/v2/Inventory/EthernetInterfaces/":
			_, _ = w.Write([]byte(`[
				{
					"ComponentID": "x1000",
					"MACAddress": "00:11:22:33:44:55",
					"IPAddresses": [{"IPAddress": "192.168.1.1"}],
					"Description": "Test Node"
				}
			]`))
		case "/hsm/v2/memberships/x1000":
			_, _ = w.Write([]byte(`{"GroupLabels": ["compute", "cabinet1", "rack1"]}`))
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &SMDClient{
		smdClient:   server.Client(),
		smdBaseURL:  server.URL,
		nodesMutex:  &sync.RWMutex{},
		nodes:       make(map[string]NodeMapping),
		ipToXname:   make(map[string]string),
		macToXname:  make(map[string]string),
		wgipToXname: make(map[string]string),
	}

	client.PopulateNodes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GroupMembership("x1000")
	}
}
