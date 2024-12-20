package wgtunnel

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// PublicKeyRequest represents the JSON payload for a WireGuard public key.
type PublicKeyRequest struct {
	PublicKey string `json:"public_key"`
}

// addClientHandler handles adding a WireGuard client.
func AddClientHandler(st Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
			return
		}

		var req PublicKeyRequest
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		publicKey := strings.TrimSpace(req.PublicKey)
		if publicKey == "" {
			http.Error(w, "Public key is required", http.StatusBadRequest)
			return
		}

		// Check for the standard header
		clientIP := r.Header.Get("X-Forwarded-For")
		clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0]) // Use the first IP in the list if multiple exist.
		// If the standard header is not found, check for the RemoteAddr
		if clientIP == "" {
			clientIP = strings.Split(r.RemoteAddr, ":")[0] // Get the client IP from the remote address and strip the port
		}
		if clientIP == "" {
			http.Error(w, "Client IP not found in request headers", http.StatusBadRequest)
			return
		}

		if net.ParseIP(clientIP) == nil {
			http.Error(w, "Invalid client IP", http.StatusBadRequest)
			return
		}

		log.Info().Msgf("Received request: PublicKey=%s, ClientIP=%s\n", publicKey, clientIP)

		// Assign a unique IP for the client.
		clientVPNIP := st.IpForPeer(clientIP, publicKey)
		if clientVPNIP == "" {
			http.Error(w, "Failed to allocate client IP", http.StatusInternalServerError)
			return
		}

		// Add the client to the WireGuard configuration.
		log.Debug().Msgf("Adding WireGuard peer: PublicKey=%s, ClientVPNIP=%s, ClientIP=%s\n", publicKey, clientVPNIP, clientIP)
		if err := AddWireGuardPeer(st.GetInterfaceName(), publicKey, clientVPNIP, clientIP); err != nil {
			http.Error(w, "Failed to configure WireGuard tunnel: "+err.Error(), http.StatusInternalServerError)
			return
		}

		serverConfig, err := st.GetServerConfig()
		if err != nil {
			http.Error(w, "Failed to get server configuration: "+err.Error(), http.StatusInternalServerError)
		}

		response := map[string]string{
			"message":           "WireGuard tunnel created successfully",
			"client-vpn-ip":     clientVPNIP,
			"server-public-key": serverConfig.PublicKey,
			"server-ip":         serverConfig.IP,
			"server-port":       serverConfig.Port,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error().Err(err).Msg("Failed to encode response")
		}
	}
}
