package smdclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPopulateNodes(t *testing.T) {
	// Mock SMD server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/hsm/v2/Inventory/EthernetInterfaces/", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// ignoring error on Write
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
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create SMDClient
	client := &SMDClient{
		smdClient:         server.Client(),
		smdBaseURL:        server.URL,
		nodesMutex:        &sync.Mutex{},
		nodes_last_update: time.Now(),
		nodes:             make(map[string]NodeMapping),
	}

	// Call PopulateNodes
	client.PopulateNodes()

	// Verify nodes map
	client.nodesMutex.Lock()
	t.Log(client.nodes)
	defer client.nodesMutex.Unlock()

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
}
func TestIPfromID(t *testing.T) {
	// Mock SMD server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/hsm/v2/Inventory/EthernetInterfaces/", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// ignoring error on Write
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
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create SMDClient
	client := &SMDClient{
		smdClient:         server.Client(),
		smdBaseURL:        server.URL,
		nodesMutex:        &sync.Mutex{},
		nodes_last_update: time.Now(),
		nodes:             make(map[string]NodeMapping),
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
		assert.Equal(t, "/hsm/v2/Inventory/EthernetInterfaces/", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// ignoring error on Write
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
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create SMDClient
	client := &SMDClient{
		smdClient:         server.Client(),
		smdBaseURL:        server.URL,
		nodesMutex:        &sync.Mutex{},
		nodes_last_update: time.Now(),
		nodes:             make(map[string]NodeMapping),
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
		assert.Equal(t, "/hsm/v2/Inventory/EthernetInterfaces/", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// ignoring error on Write
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
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create SMDClient
	client := &SMDClient{
		smdClient:         server.Client(),
		smdBaseURL:        server.URL,
		nodesMutex:        &sync.Mutex{},
		nodes_last_update: time.Now(),
		nodes:             make(map[string]NodeMapping),
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
