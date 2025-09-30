package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// WireGuardMiddlewareWithProxy creates a middleware to enforce WireGuard policy.
func WireGuardMiddlewareWithProxy(wireGuardCIDR string, allow bool) func(http.Handler) http.Handler {
	_, wgNet, err := net.ParseCIDR(wireGuardCIDR)
	if err != nil {
		panic("Invalid WireGuard CIDR provided: " + err.Error())
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var clientIP string

			// Check for X-Forwarded-For header
			xff := r.Header.Get("X-Forwarded-For")
			if xff != "" {
				clientIP = strings.Split(xff, ",")[0]
			}

			// Check for Forwarded header
			if clientIP == "" {
				forwarded := r.Header.Get("Forwarded")
				if forwarded != "" {
					for _, part := range strings.Split(forwarded, ";") {
						if strings.HasPrefix(strings.TrimSpace(part), "for=") {
							clientIP = strings.Trim(strings.Split(part, "=")[1], "\"")
							break
						}
					}
				}
			}

			// Fallback to RemoteAddr
			if clientIP == "" {
				clientIP, _, err = net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					http.Error(w, "Invalid Remote Address", http.StatusForbidden)
					return
				}
			}

			// Parse client IP
			ip := net.ParseIP(clientIP)
			if ip == nil {
				http.Error(w, "Invalid IP Address", http.StatusForbidden)
				return
			}

			// Check if IP is in WireGuard subnet
			isInWireGuardSubnet := wgNet.Contains(ip)

			// Enforce policy
			if allow && !isInWireGuardSubnet {
				http.Error(w, "Access denied: Not in WireGuard subnet", http.StatusForbidden)
				return
			}
			if !allow && isInWireGuardSubnet {
				http.Error(w, "Access denied: WireGuard traffic not allowed", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// WireGuardMiddleware enforces policies based on the interface and subnet.
func WireGuardMiddlewareWithInterface(wireGuardInterface string, wireGuardCIDR string) func(http.Handler) http.Handler {
	// Parse the WireGuard CIDR into a *net.IPNet
	_, wgNet, err := net.ParseCIDR(wireGuardCIDR)
	if err != nil {
		panic("Invalid WireGuard CIDR provided: " + err.Error())
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Retrieve the local address (where the request arrived)
			localAddr := r.Context().Value(http.LocalAddrContextKey)
			if localAddr == nil {
				log.Debug().Msg("Could not determine local address")
				http.Error(w, "Could not determine local address", http.StatusForbidden)
				return
			}

			// Assert net.Addr from the context
			addr, ok := localAddr.(net.Addr)
			if !ok {
				log.Debug().Msg("Invalid address format")
				http.Error(w, "Invalid address format", http.StatusForbidden)
				return
			}

			// Extract the local IP
			localIP, _, err := net.SplitHostPort(addr.String())
			if err != nil {
				log.Debug().Msg("Could not extract IP from address")
				http.Error(w, "Could not extract IP from address", http.StatusForbidden)
				return
			}

			ip := net.ParseIP(localIP)
			if ip == nil {
				log.Debug().Msg("Invalid IP Address")
				http.Error(w, "Invalid IP Address", http.StatusForbidden)
				return
			}

			var clientIP string

			// Check for X-Forwarded-For header
			xff := r.Header.Get("X-Forwarded-For")
			if xff != "" {
				clientIP = strings.Split(xff, ",")[0]
			}

			// Check for Forwarded header
			if clientIP == "" {
				forwarded := r.Header.Get("Forwarded")
				if forwarded != "" {
					for _, part := range strings.Split(forwarded, ";") {
						if strings.HasPrefix(strings.TrimSpace(part), "for=") {
							clientIP = strings.Trim(strings.Split(part, "=")[1], "\"")
							break
						}
					}
				}
			}

			// Fallback to RemoteAddr
			if clientIP == "" {
				clientIP, _, err = net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					http.Error(w, "Invalid Remote Address", http.StatusForbidden)
					return
				}
			}

			// Check if the IP matches the WireGuard subnet
			isInWireGuardSubnet := wgNet.Contains(ip)
			var recievedInterface net.Interface

			// Check if the request arrived on the WireGuard interface
			isOnWireGuardInterface := false
			interfaces, err := net.Interfaces()
			if err != nil {
				log.Debug().Msg("Could not retrieve network interfaces")
				http.Error(w, "Could not retrieve network interfaces", http.StatusInternalServerError)
				return
			}
			for _, iface := range interfaces {
				addrs, _ := iface.Addrs() // Ignoring error on Addrs() as we can still check other interfaces
				for _, ifaceAddr := range addrs {
					if ipNet, ok := ifaceAddr.(*net.IPNet); ok && ipNet.IP.Equal(ip) {
						recievedInterface = iface
						if iface.Name == wireGuardInterface {
							isOnWireGuardInterface = true
							break
						}
					}
				}
			}

			log.Debug().
				Str("localIP", localIP).
				Str("clientIP", clientIP).
				Str("interface", recievedInterface.Name).
				Bool("isInWireGuardSubnet", isInWireGuardSubnet).
				Bool("isOnWireGuardInterface", isOnWireGuardInterface).
				Msg("WireGuard policy check")

			// Enforce the policy: deny if neither condition is true
			if !isInWireGuardSubnet && !isOnWireGuardInterface {
				log.Debug().Msgf("Access denied: IP %s not in WireGuard subnet or interface", localIP)
				http.Error(w, fmt.Sprintf("Access denied: IP %s not in WireGuard subnet or interface", localIP), http.StatusForbidden)
				return
			}

			// Pass the request to the next middleware or handler
			next.ServeHTTP(w, r)
		})
	}
}
