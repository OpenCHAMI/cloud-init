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

// WireGuardMiddlewareWithInterface enforces policies based on the client's IP and the WireGuard subnet.
// It allows requests if the CLIENT IP is either:
// 1. In the WireGuard subnet (e.g., 100.97.0.0/16), OR
// 2. Arriving on the specified WireGuard interface
//
// This ensures that nodes can access cloud-init either:
// - Through their WireGuard tunnel (client IP in WireGuard subnet)
// - Directly on the server's WireGuard interface
func WireGuardMiddlewareWithInterface(wireGuardInterface string, wireGuardCIDR string) func(http.Handler) http.Handler {
	// Parse the WireGuard CIDR into a *net.IPNet
	_, wgNet, err := net.ParseCIDR(wireGuardCIDR)
	if err != nil {
		panic("Invalid WireGuard CIDR provided: " + err.Error())
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP from request
			var clientIP string

			// Check for X-Forwarded-For header
			xff := r.Header.Get("X-Forwarded-For")
			if xff != "" {
				clientIP = strings.TrimSpace(strings.Split(xff, ",")[0])
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
				var err error
				clientIP, _, err = net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					log.Debug().Err(err).Msg("Invalid Remote Address")
					http.Error(w, "Invalid Remote Address", http.StatusForbidden)
					return
				}
			}

			// Parse client IP to check if it's in WireGuard subnet
			clientIPParsed := net.ParseIP(clientIP)
			if clientIPParsed == nil {
				http.Error(w, "Invalid client IP Address", http.StatusForbidden)
				return
			}

			// Check if the CLIENT IP is in the WireGuard subnet
			isInWireGuardSubnet := wgNet.Contains(clientIPParsed)
			var recievedInterface net.Interface

			// Check if the request arrived on the WireGuard interface
			isOnWireGuardInterface := false
			interfaces, err := net.Interfaces()
			if err != nil {
				log.Debug().Msg("Could not retrieve network interfaces")
				http.Error(w, "Could not retrieve network interfaces", http.StatusInternalServerError)
				return
			}

			// Check if CLIENT IP is in WireGuard subnet
			isInWireGuardSubnet := wgNet.Contains(clientIPParsed)

			// Retrieve the local address (where the request arrived on the server)
			var localIP string
			var isOnWireGuardInterface bool
			var receivedInterface string

			localAddr := r.Context().Value(http.LocalAddrContextKey)
			if localAddr != nil {
				if addr, ok := localAddr.(net.Addr); ok {
					localIP, _, _ = net.SplitHostPort(addr.String())
					if localIPParsed := net.ParseIP(localIP); localIPParsed != nil {
						// Check if the request arrived on the WireGuard interface
						interfaces, err := net.Interfaces()
						if err == nil {
							for _, iface := range interfaces {
								addrs, _ := iface.Addrs() // Ignoring error on Addrs() as we can still check other interfaces
								for _, ifaceAddr := range addrs {
									if ipNet, ok := ifaceAddr.(*net.IPNet); ok && ipNet.IP.Equal(localIPParsed) {
										receivedInterface = iface.Name
										if iface.Name == wireGuardInterface {
											isOnWireGuardInterface = true
											break
										}
									}
								}
								if isOnWireGuardInterface {
									break
								}
							}
						}
					}
				}
			}

			log.Debug().
				Str("localIP", localIP).
				Str("clientIP", clientIP).
				Str("interface", receivedInterface).
				Bool("isInWireGuardSubnet", isInWireGuardSubnet).
				Bool("isOnWireGuardInterface", isOnWireGuardInterface).
				Msg("WireGuard policy check")

			// Enforce the policy: allow if CLIENT IP is in WireGuard subnet OR request arrived on WireGuard interface
			if !isInWireGuardSubnet && !isOnWireGuardInterface {
				log.Debug().
					Str("clientIP", clientIP).
					Str("localIP", localIP).
					Str("interface", receivedInterface).
					Msgf("Access denied: client IP %s not in WireGuard subnet and request not on WireGuard interface", clientIP)
				http.Error(w, fmt.Sprintf("Access denied: client IP %s not in WireGuard subnet or on WireGuard interface", clientIP), http.StatusForbidden)
				return
			}

			// Pass the request to the next middleware or handler
			next.ServeHTTP(w, r)
		})
	}
}
