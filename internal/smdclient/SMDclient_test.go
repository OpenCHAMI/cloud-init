package smdclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPopulateNodes(t *testing.T) {
	var requestsMutex sync.Mutex
	requests := make([]string, 0, 2)

	// Mock SMD server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestsMutex.Lock()
		requests = append(requests, r.URL.RequestURI())
		requestsMutex.Unlock()

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
				"MACAddress": "66:77:88:99:AA:BB",
				"IPAddresses": [{"IPAddress": "192.168.1.2"}, {"IPAddr": "192.168.1.3"}],
				"Description": "Test Node 2"
			},
			{
				"ComponentID": "x1002",
				"MACAddress": "CC:DD:EE:FF:00:11",
				"IPAddresses": [{"IPAddress": "192.168.1.4"},{"IPAddress": "192.168.1.40"}],
				"Description": "Test Node 3"
			},
			{
				"ComponentID": "x1003",	
				"MACAddress": "22:33:44:55:66:77",
				"IPAddresses": [{"IPAddress": "192.168.1.5"}],
				"Description": "Test Node 4 Interface 1"
		    },
			{
				"ComponentID": "x1003",		
				"MACAddress": "88:99:AA:BB:CC:DD",
				"IPAddresses": [{"IPAddr": "192.168.1.6"}],
				"Description": "Test Node 4 Interface 2"
			}
		]`))
		case "/hsm/v2/memberships":
			_, _ = w.Write([]byte(`[
				{"id":"x1000","groupLabels":["compute"],"partitionName":"ignored"},
				{"id":"x1001","groupLabels":["compute","io"],"partitionName":""},
				{"id":"x1002","groupLabels":["compute"],"partitionName":""},
				{"id":"x1003","groupLabels":["compute","cabinet1"],"partitionName":""},
				{"id":"x9999","groupLabels":["unrelated"],"partitionName":""}
			]`))
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create SMDClient
	client := &SMDClient{
		smdClient:         server.Client(),
		smdBaseURL:        server.URL,
		nodesMutex:        &sync.RWMutex{},
		nodes_last_update: time.Now(),
		nodes:             make(map[string]NodeMapping),
		ipToXname:         make(map[string]string),
		macToXname:        make(map[string]string),
		wgipToXname:       make(map[string]string),
	}

	// Call PopulateNodes
	client.PopulateNodes()

	// Verify nodes map
	client.nodesMutex.RLock()
	t.Log(client.nodes)
	defer client.nodesMutex.RUnlock()

	assert.Equal(t, 4, len(client.nodes))

	node1, exists := client.nodes["x1000"]
	assert.True(t, exists)
	assert.Equal(t, "x1000", node1.Xname)
	assert.Equal(t, 1, len(node1.Interfaces))
	assert.Equal(t, "00:11:22:33:44:55", node1.Interfaces[0].MAC)
	assert.Equal(t, "192.168.1.1", node1.Interfaces[0].IP)
	assert.Equal(t, "Test Node 1", node1.Interfaces[0].Desc)

	node2, exists := client.nodes["x1001"]
	assert.True(t, exists)
	assert.Equal(t, "x1001", node2.Xname)
	assert.Equal(t, 1, len(node2.Interfaces))
	assert.Equal(t, "66:77:88:99:AA:BB", node2.Interfaces[0].MAC)
	assert.Equal(t, "192.168.1.2", node2.Interfaces[0].IP)
	assert.Equal(t, "Test Node 2", node2.Interfaces[0].Desc)

	node3, exists := client.nodes["x1002"]
	assert.True(t, exists)
	assert.Equal(t, "x1002", node3.Xname)
	assert.Equal(t, 1, len(node3.Interfaces))

	node4, exists := client.nodes["x1003"]
	assert.True(t, exists)
	assert.Equal(t, "x1003", node4.Xname)
	assert.Equal(t, 2, len(node4.Interfaces))
	assert.Equal(t, []string{"compute", "cabinet1"}, node4.Groups)

	requestsMutex.Lock()
	defer requestsMutex.Unlock()
	require.Equal(t, []string{
		"/hsm/v2/Inventory/EthernetInterfaces/",
		"/hsm/v2/memberships?type=node",
	}, requests)
	for _, request := range requests {
		assert.False(t, strings.HasPrefix(request, "/hsm/v2/memberships/"), "unexpected per-node membership request: %s", request)
	}
}
func TestIPfromID(t *testing.T) {
	// Mock SMD server
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
				"MACAddress": "66:77:88:99:AA:BB",
				"IPAddresses": [{"IPAddress": "192.168.1.2"}, {"IPAddr": "192.168.1.3"}],
				"Description": "Test Node 2"
			},
			{
				"ComponentID": "x1002",
				"MACAddress": "CC:DD:EE:FF:00:11",
				"IPAddresses": [{"IPAddress": "192.168.1.4"},{"IPAddress": "192.168.1.40"}],
				"Description": "Test Node 3"
			},
			{
				"ComponentID": "x1003",	
				"MACAddress": "22:33:44:55:66:77",
				"IPAddresses": [{"IPAddress": "192.168.1.5"}],
				"Description": "Test Node 4 Interface 1"
			},
			{
				"ComponentID": "x1003",		
				"MACAddress": "88:99:AA:BB:CC:DD",
				"IPAddresses": [{"IPAddr": "192.168.1.6"}],
				"Description": "Test Node 4 Interface 2"
			}
		]`))
		case "/hsm/v2/memberships":
			_, _ = w.Write([]byte(`[
				{"id":"x1000","groupLabels":["compute"],"partitionName":""},
				{"id":"x1001","groupLabels":["compute"],"partitionName":""},
				{"id":"x1002","groupLabels":["compute"],"partitionName":""},
				{"id":"x1003","groupLabels":["compute"],"partitionName":""}
			]`))
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create SMDClient
	client := &SMDClient{
		smdClient:         server.Client(),
		smdBaseURL:        server.URL,
		nodesMutex:        &sync.RWMutex{},
		nodes_last_update: time.Now(),
		nodes:             make(map[string]NodeMapping),
		ipToXname:         make(map[string]string),
		macToXname:        make(map[string]string),
		wgipToXname:       make(map[string]string),
	}

	// Call PopulateNodes to populate the nodes map
	client.PopulateNodes()

	// Test cases
	tests := []struct {
		id       string
		expected string
		err      error
	}{
		{"x1000", "192.168.1.1", nil},
		{"x1001", "192.168.1.2", nil},
		{"x1002", "192.168.1.4", nil},
		{"x1003", "192.168.1.5", nil},
		{"x9999", "", errors.New("ID x9999 not found in nodes")},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			ip, err := client.IPfromID(tt.id)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, ip)
			}
		})
	}
}
func TestIDfromIP(t *testing.T) {
	// Mock SMD server
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
				"MACAddress": "66:77:88:99:AA:BB",
				"IPAddresses": [{"IPAddress": "192.168.1.2"}, {"IPAddr": "192.168.1.3"}],
				"Description": "Test Node 2"
			},
			{
				"ComponentID": "x1002",
				"MACAddress": "CC:DD:EE:FF:00:11",
				"IPAddresses": [{"IPAddress": "192.168.1.4"},{"IPAddress": "192.168.1.40"}],
				"Description": "Test Node 3"
			},
			{
				"ComponentID": "x1003",	
				"MACAddress": "22:33:44:55:66:77",
				"IPAddresses": [{"IPAddress": "192.168.1.5"}],
				"Description": "Test Node 4 Interface 1"
			},
			{
				"ComponentID": "x1003",		
				"MACAddress": "88:99:AA:BB:CC:DD",
				"IPAddresses": [{"IPAddr": "192.168.1.6"}],
				"Description": "Test Node 4 Interface 2"
			}
		]`))
		case "/hsm/v2/memberships":
			_, _ = w.Write([]byte(`[
				{"id":"x1000","groupLabels":["compute"],"partitionName":""},
				{"id":"x1001","groupLabels":["compute"],"partitionName":""},
				{"id":"x1002","groupLabels":["compute"],"partitionName":""},
				{"id":"x1003","groupLabels":["compute"],"partitionName":""}
			]`))
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create SMDClient
	client := &SMDClient{
		smdClient:         server.Client(),
		smdBaseURL:        server.URL,
		nodesMutex:        &sync.RWMutex{},
		nodes_last_update: time.Now(),
		nodes:             make(map[string]NodeMapping),
		ipToXname:         make(map[string]string),
		macToXname:        make(map[string]string),
		wgipToXname:       make(map[string]string),
	}

	// Call PopulateNodes to populate the nodes map
	client.PopulateNodes()

	// Test cases
	tests := []struct {
		ip       string
		expected string
		err      error
	}{
		{"192.168.1.1", "x1000", nil},
		{"192.168.1.2", "x1001", nil},
		{"192.168.1.4", "x1002", nil},
		{"192.168.1.5", "x1003", nil},
		{"192.168.1.99", "", errors.New("IP address 192.168.1.99 not found for an xname in nodes")},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			id, err := client.IDfromIP(tt.ip)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, id)
			}
		})
	}
}
func TestIDfromMAC(t *testing.T) {
	// Mock SMD server
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
				"MACAddress": "66:77:88:99:AA:BB",
				"IPAddresses": [{"IPAddress": "192.168.1.2"}, {"IPAddr": "192.168.1.3"}],
				"Description": "Test Node 2"
			},
			{
				"ComponentID": "x1002",
				"MACAddress": "CC:DD:EE:FF:00:11",
				"IPAddresses": [{"IPAddress": "192.168.1.4"},{"IPAddress": "192.168.1.40"}],
				"Description": "Test Node 3"
			},
			{
				"ComponentID": "x1003",	
				"MACAddress": "22:33:44:55:66:77",
				"IPAddresses": [{"IPAddress": "192.168.1.5"}],
				"Description": "Test Node 4 Interface 1"
			},
			{
				"ComponentID": "x1003",		
				"MACAddress": "88:99:AA:BB:CC:DD",
				"IPAddresses": [{"IPAddr": "192.168.1.6"}],
				"Description": "Test Node 4 Interface 2"
			}
		]`))
		case "/hsm/v2/memberships":
			_, _ = w.Write([]byte(`[
				{"id":"x1000","groupLabels":["compute"],"partitionName":""},
				{"id":"x1001","groupLabels":["compute"],"partitionName":""},
				{"id":"x1002","groupLabels":["compute"],"partitionName":""},
				{"id":"x1003","groupLabels":["compute"],"partitionName":""}
			]`))
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create SMDClient
	client := &SMDClient{
		smdClient:         server.Client(),
		smdBaseURL:        server.URL,
		nodesMutex:        &sync.RWMutex{},
		nodes_last_update: time.Now(),
		nodes:             make(map[string]NodeMapping),
		ipToXname:         make(map[string]string),
		macToXname:        make(map[string]string),
		wgipToXname:       make(map[string]string),
	}

	// Call PopulateNodes to populate the nodes map
	client.PopulateNodes()

	// Test cases
	tests := []struct {
		mac      string
		expected string
		err      error
	}{
		{"00:11:22:33:44:55", "x1000", nil},
		{"66:77:88:99:AA:BB", "x1001", nil},
		{"CC:DD:EE:FF:00:11", "x1002", nil},
		{"22:33:44:55:66:77", "x1003", nil},
		{"FF:FF:FF:FF:FF:FF", "", errors.New("MAC FF:FF:FF:FF:FF:FF not found for an xname in nodes")},
	}

	for _, tt := range tests {
		t.Run(tt.mac, func(t *testing.T) {
			id, err := client.IDfromMAC(tt.mac)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, id)
			}
		})
	}
}

func TestPopulateNodesMissingMembershipUsesEmptyGroups(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/hsm/v2/Inventory/EthernetInterfaces/":
			_, _ = w.Write([]byte(`[
				{"ComponentID":"x1000","MACAddress":"00:11:22:33:44:55","IPAddresses":[{"IPAddress":"192.168.1.1"}]},
				{"ComponentID":"x1001","MACAddress":"00:11:22:33:44:66","IPAddresses":[{"IPAddress":"192.168.1.2"}]}
			]`))
		case "/hsm/v2/memberships":
			if got := r.URL.Query().Get("type"); got != "node" {
				t.Errorf("membership type query = %q, want node", got)
			}
			_, _ = w.Write([]byte(`[{"id":"x1000","groupLabels":["compute"],"partitionName":"ignored"}]`))
		default:
			t.Errorf("unexpected SMD request: %s", r.URL.RequestURI())
			w.WriteHeader(http.StatusNotFound)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := newTestSMDClient(server)
	client.PopulateNodes()

	groups, err := client.GroupMembership("x1001")
	require.NoError(t, err)
	assert.Empty(t, groups)
	assert.NotNil(t, groups)
}

func TestPopulateNodesBulkMembershipFailurePreservesCache(t *testing.T) {
	tests := []struct {
		name         string
		writeFailure func(http.ResponseWriter)
	}{
		{
			name: "HTTP failure",
			writeFailure: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"error":"unavailable"}`))
			},
		},
		{
			name: "malformed JSON",
			writeFailure: func(w http.ResponseWriter) {
				_, _ = w.Write([]byte(`[{"id":"x1000","groupLabels":`))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var failMemberships atomic.Bool
			var perNodeRequests atomic.Int32
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/hsm/v2/Inventory/EthernetInterfaces/":
					_, _ = w.Write([]byte(`[{"ComponentID":"x1000","MACAddress":"00:11:22:33:44:55","IPAddresses":[{"IPAddress":"192.168.1.1"}]}]`))
				case "/hsm/v2/memberships":
					if failMemberships.Load() {
						tt.writeFailure(w)
						return
					}
					_, _ = w.Write([]byte(`[{"id":"x1000","groupLabels":["compute"],"partitionName":""}]`))
				default:
					if strings.HasPrefix(r.URL.Path, "/hsm/v2/memberships/") {
						perNodeRequests.Add(1)
					}
					w.WriteHeader(http.StatusNotFound)
				}
			})
			server := httptest.NewServer(handler)
			defer server.Close()

			client := newTestSMDClient(server)
			client.PopulateNodes()
			require.NoError(t, client.AddWGIP("x1000", "10.99.0.1"))

			client.nodesMutex.RLock()
			oldTimestamp := client.nodes_last_update
			client.nodesMutex.RUnlock()
			failMemberships.Store(true)
			client.PopulateNodes()

			groups, err := client.GroupMembership("x1000")
			require.NoError(t, err)
			assert.Equal(t, []string{"compute"}, groups)
			assert.Equal(t, "x1000", mustIDfromIP(t, client, "192.168.1.1"))
			assert.Equal(t, "x1000", mustIDfromIP(t, client, "10.99.0.1"))
			assert.Equal(t, "x1000", mustIDfromMAC(t, client, "00:11:22:33:44:55"))
			wgip, err := client.WGIPfromID("x1000")
			require.NoError(t, err)
			assert.Equal(t, "10.99.0.1", wgip)
			client.nodesMutex.RLock()
			assert.Equal(t, oldTimestamp, client.nodes_last_update)
			client.nodesMutex.RUnlock()
			assert.Zero(t, perNodeRequests.Load())
		})
	}
}

func newTestSMDClient(server *httptest.Server) *SMDClient {
	return &SMDClient{
		smdClient:   server.Client(),
		smdBaseURL:  server.URL,
		nodesMutex:  &sync.RWMutex{},
		nodes:       make(map[string]NodeMapping),
		ipToXname:   make(map[string]string),
		macToXname:  make(map[string]string),
		wgipToXname: make(map[string]string),
	}
}

func mustIDfromIP(t *testing.T, client *SMDClient, ip string) string {
	t.Helper()
	id, err := client.IDfromIP(ip)
	require.NoError(t, err)
	return id
}

func mustIDfromMAC(t *testing.T, client *SMDClient, mac string) string {
	t.Helper()
	id, err := client.IDfromMAC(mac)
	require.NoError(t, err)
	return id
}
