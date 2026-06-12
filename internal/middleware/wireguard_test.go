package middleware

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockAddr implements net.Addr interface for testing
type mockAddr struct {
	network string
	address string
}

func (m mockAddr) Network() string {
	return m.network
}

func (m mockAddr) String() string {
	return m.address
}

// TestWireGuardMiddlewareWithProxy tests the proxy-based WireGuard middleware
func TestWireGuardMiddlewareWithProxy(t *testing.T) {
	testCases := []struct {
		name           string
		wireGuardCIDR  string
		allow          bool
		clientIP       string
		xff            string
		forwarded      string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Allow client in WireGuard subnet",
			wireGuardCIDR:  "100.97.0.0/16",
			allow:          true,
			clientIP:       "100.97.0.5",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Deny client not in WireGuard subnet when allow=true",
			wireGuardCIDR:  "100.97.0.0/16",
			allow:          true,
			clientIP:       "192.168.1.10",
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Access denied: Not in WireGuard subnet\n",
		},
		{
			name:           "Allow client not in WireGuard subnet when allow=false",
			wireGuardCIDR:  "100.97.0.0/16",
			allow:          false,
			clientIP:       "192.168.1.10",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Deny client in WireGuard subnet when allow=false",
			wireGuardCIDR:  "100.97.0.0/16",
			allow:          false,
			clientIP:       "100.97.0.5",
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Access denied: WireGuard traffic not allowed\n",
		},
		{
			name:           "Use X-Forwarded-For header",
			wireGuardCIDR:  "100.97.0.0/16",
			allow:          true,
			clientIP:       "192.168.1.10", // RemoteAddr (should be ignored)
			xff:            "100.97.0.20",  // X-Forwarded-For (should be used)
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Use Forwarded header",
			wireGuardCIDR:  "100.97.0.0/16",
			allow:          true,
			clientIP:       "192.168.1.10",    // RemoteAddr (should be ignored)
			forwarded:      "for=100.97.0.30", // Forwarded (should be used)
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "X-Forwarded-For takes precedence over Forwarded",
			wireGuardCIDR:  "100.97.0.0/16",
			allow:          true,
			clientIP:       "192.168.1.10",
			xff:            "100.97.0.40",
			forwarded:      "for=192.168.1.20",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test handler that will be wrapped by the middleware
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			})

			// Apply middleware
			middleware := WireGuardMiddlewareWithProxy(tc.wireGuardCIDR, tc.allow)
			wrappedHandler := middleware(handler)

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tc.clientIP + ":12345"
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.forwarded != "" {
				req.Header.Set("Forwarded", tc.forwarded)
			}

			// Record response
			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rr.Code)
			}

			// Check body
			if rr.Body.String() != tc.expectedBody {
				t.Errorf("Expected body %q, got %q", tc.expectedBody, rr.Body.String())
			}
		})
	}
}

// TestWireGuardMiddlewareWithInterface tests the interface-based WireGuard middleware
func TestWireGuardMiddlewareWithInterface(t *testing.T) {
	testCases := []struct {
		name               string
		wireGuardCIDR      string
		wireGuardInterface string
		clientIP           string
		xff                string
		localAddr          net.Addr
		expectedStatus     int
		description        string
	}{
		{
			name:               "Allow client in WireGuard subnet",
			wireGuardCIDR:      "100.97.0.0/16",
			wireGuardInterface: "wg0",
			clientIP:           "100.97.0.5",
			localAddr:          mockAddr{"tcp", "192.168.1.100:27777"},
			expectedStatus:     http.StatusOK,
			description:        "Client with WireGuard IP should be allowed regardless of interface",
		},
		{
			name:               "Deny client not in WireGuard subnet on non-WireGuard interface",
			wireGuardCIDR:      "100.97.0.0/16",
			wireGuardInterface: "wg0",
			clientIP:           "192.168.1.10",
			localAddr:          mockAddr{"tcp", "192.168.1.100:27777"},
			expectedStatus:     http.StatusForbidden,
			description:        "Client without WireGuard IP on regular interface should be denied",
		},
		{
			name:               "Allow client with X-Forwarded-For in WireGuard subnet",
			wireGuardCIDR:      "100.97.0.0/16",
			wireGuardInterface: "wg0",
			clientIP:           "192.168.1.10", // RemoteAddr
			xff:                "100.97.0.20",  // X-Forwarded-For
			localAddr:          mockAddr{"tcp", "192.168.1.100:27777"},
			expectedStatus:     http.StatusOK,
			description:        "Should use X-Forwarded-For when present",
		},
		{
			name:               "Allow multiple IPs in X-Forwarded-For (use first)",
			wireGuardCIDR:      "100.97.0.0/16",
			wireGuardInterface: "wg0",
			clientIP:           "192.168.1.10",
			xff:                "100.97.0.30, 10.0.0.1, 172.16.0.1",
			localAddr:          mockAddr{"tcp", "192.168.1.100:27777"},
			expectedStatus:     http.StatusOK,
			description:        "Should use first IP in X-Forwarded-For chain",
		},
		{
			name:               "Deny invalid client IP",
			wireGuardCIDR:      "100.97.0.0/16",
			wireGuardInterface: "wg0",
			clientIP:           "invalid-ip",
			localAddr:          mockAddr{"tcp", "192.168.1.100:27777"},
			expectedStatus:     http.StatusForbidden,
			description:        "Invalid IP should be rejected",
		},
		{
			name:               "Allow client at edge of subnet",
			wireGuardCIDR:      "100.97.0.0/16",
			wireGuardInterface: "wg0",
			clientIP:           "100.97.255.255",
			localAddr:          mockAddr{"tcp", "192.168.1.100:27777"},
			expectedStatus:     http.StatusOK,
			description:        "IP at edge of subnet should be allowed",
		},
		{
			name:               "Deny client just outside subnet",
			wireGuardCIDR:      "100.97.0.0/16",
			wireGuardInterface: "wg0",
			clientIP:           "100.98.0.1",
			localAddr:          mockAddr{"tcp", "192.168.1.100:27777"},
			expectedStatus:     http.StatusForbidden,
			description:        "IP just outside subnet should be denied",
		},
		{
			name:               "Allow client with /24 subnet",
			wireGuardCIDR:      "10.89.0.0/24",
			wireGuardInterface: "wg0",
			clientIP:           "10.89.0.50",
			localAddr:          mockAddr{"tcp", "192.168.1.100:27777"},
			expectedStatus:     http.StatusOK,
			description:        "Should work with smaller subnets",
		},
		{
			name:               "Handle no local address gracefully",
			wireGuardCIDR:      "100.97.0.0/16",
			wireGuardInterface: "wg0",
			clientIP:           "100.97.0.5",
			localAddr:          nil,
			expectedStatus:     http.StatusOK,
			description:        "Should still allow based on client IP when local addr is missing",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			})

			// Apply middleware
			middleware := WireGuardMiddlewareWithInterface(tc.wireGuardInterface, tc.wireGuardCIDR)
			wrappedHandler := middleware(handler)

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tc.clientIP + ":12345"
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}

			// Add local address to context if provided
			if tc.localAddr != nil {
				ctx := context.WithValue(req.Context(), http.LocalAddrContextKey, tc.localAddr)
				req = req.WithContext(ctx)
			}

			// Record response
			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tc.expectedStatus {
				t.Errorf("%s: Expected status %d, got %d", tc.description, tc.expectedStatus, rr.Code)
			}
		})
	}
}

// TestWireGuardMiddlewareWithProxy_InvalidCIDR tests panic on invalid CIDR
func TestWireGuardMiddlewareWithProxy_InvalidCIDR(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic with invalid CIDR, but didn't panic")
		}
	}()

	// This should panic
	_ = WireGuardMiddlewareWithProxy("invalid-cidr", true)
}

// TestWireGuardMiddlewareWithInterface_InvalidCIDR tests panic on invalid CIDR
func TestWireGuardMiddlewareWithInterface_InvalidCIDR(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic with invalid CIDR, but didn't panic")
		}
	}()

	// This should panic
	_ = WireGuardMiddlewareWithInterface("wg0", "invalid-cidr")
}

// TestWireGuardMiddleware_RealWorldScenario simulates the bug scenario from v1.4.1
func TestWireGuardMiddleware_RealWorldScenario(t *testing.T) {
	// Scenario: 200 nodes trying to get cloud-init data
	// - WireGuard subnet: 100.97.0.0/16
	// - Nodes have regular IPs: 192.168.1.0/24
	// - Some nodes have established WireGuard tunnels and got IPs in 100.97.0.0/16
	// - Server listens on 192.168.1.100:27777

	middleware := WireGuardMiddlewareWithInterface("wg0", "100.97.0.0/16")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("cloud-init-data"))
	})
	wrappedHandler := middleware(handler)

	testCases := []struct {
		name           string
		nodeIP         string
		hasWireGuard   bool
		wgIP           string
		expectedStatus int
		description    string
	}{
		{
			name:           "Node with WireGuard tunnel",
			nodeIP:         "192.168.1.10",
			hasWireGuard:   true,
			wgIP:           "100.97.0.5",
			expectedStatus: http.StatusOK,
			description:    "Node that established WireGuard tunnel should be allowed",
		},
		{
			name:           "Node without WireGuard tunnel",
			nodeIP:         "192.168.1.11",
			hasWireGuard:   false,
			expectedStatus: http.StatusForbidden,
			description:    "Node without WireGuard tunnel should be denied",
		},
		{
			name:           "Node with WireGuard at edge of subnet",
			nodeIP:         "192.168.1.12",
			hasWireGuard:   true,
			wgIP:           "100.97.255.254",
			expectedStatus: http.StatusOK,
			description:    "Node with WireGuard IP at edge should be allowed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/user-data", nil)

			// Simulate the request coming from the node
			if tc.hasWireGuard {
				// Node uses WireGuard IP as source
				req.RemoteAddr = tc.wgIP + ":54321"
			} else {
				// Node uses regular IP as source
				req.RemoteAddr = tc.nodeIP + ":54321"
			}

			// Server received request on its eth0 interface
			ctx := context.WithValue(req.Context(), http.LocalAddrContextKey, mockAddr{"tcp", "192.168.1.100:27777"})
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("%s: Expected status %d, got %d. Body: %s",
					tc.description, tc.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}

// BenchmarkWireGuardMiddlewareWithInterface benchmarks the middleware performance
func BenchmarkWireGuardMiddlewareWithInterface(b *testing.B) {
	middleware := WireGuardMiddlewareWithInterface("wg0", "100.97.0.0/16")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "100.97.0.5:12345"
	ctx := context.WithValue(req.Context(), http.LocalAddrContextKey, mockAddr{"tcp", "192.168.1.100:27777"})
	req = req.WithContext(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
	}
}
