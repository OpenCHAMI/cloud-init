package wgtunnel

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/rs/zerolog/log"
)

// PublicKeyRequest represents the JSON payload for a WireGuard public key.
type PublicKeyRequest struct {
	PublicKey string `json:"public_key" yaml:"public_key" example:"9NS6+NR0J38SZ9IlY9hBDLs6aBpNDhxHUHL8OTlNEDU=" description:"WireGuard public key content"`
}

// WGResponse represents the JSON payload for a response from the WireGuard
// server.
type WGResponse struct {
	Message      string `json:"message" yaml:"message" example:"WireGuard tunnel created successfully"`
	ClientVPNIP  string `json:"client-vpn-ip" yaml:"client-vpn-ip" example:"10.89.0.7" description:"Assigned WireGuard VPN IP address"`
	ServerPubKey string `json:"server-public-key" yaml:"server-public-key" example:"dHMOGL8vTGhTgqXyYdu6cLGXEPmTcWm+vS18GcQseyg="`
	ServerIP     string `json:"server-ip" yaml:"server-ip" example:"10.87.0.1" description:"WireGuard server IP"`
	ServerPort   string `json:"server-port" yaml:"server-port" example:"51820" description:"WireGuard server port"`
}

// AddClientHandler godoc
//
//	@Summary		Add a WireGuard client
//	@Description	Initiate a WireGuard tunnel from a client using its public key
//	@Description	and peer name (IP address).
//	@Description
//	@Description	The source IP of the request is read and is used as the peer
//	@Description	name along with the public key to authenticate unless the
//	@Description	`X-Forward-For` header is set. In that case, the value of the
//	@Description	header is used as the peer name. If the peer exists in the
//	@Description	internal tunnel manager, the IP presented is the one used.
//	@Description	Otherwise, the next available IP in range is assigned.
//	@Accept			json
//	@Produce		json
//	@Success		200				{object}	WGResponse
//	@Failure		400				{object}	nil
//	@Failure		500				{object}	nil
//	@Param			pubkey			body		PublicKeyRequest	true	"WireGuard public key of client"
//	@Param			X-Forwarded-For	header		string				false	"Override source IP"
//	@Router			/cloud-init/wg-init [post]
func AddClientHandler(im *InterfaceManager, smdClient smdclient.SMDClientInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
			return
		}

		var req PublicKeyRequest
		defer func() {
			_ = r.Body.Close()
		}()
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
		clientVPNIP := im.IpForPeer(clientIP, publicKey)
		if clientVPNIP == "" {
			http.Error(w, "Failed to allocate client IP", http.StatusInternalServerError)
			return
		}

		// Add the wireguard ip to the SMD client
		id, err := smdClient.IDfromIP(clientIP)
		if err != nil {
			http.Error(w, "Failed to get ID from IP through our SMD client: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := smdClient.AddWGIP(id, clientVPNIP); err != nil {
			http.Error(w, fmt.Sprintf("Failed to add WireGuard IP to SMD client as %s : ", id)+err.Error(), http.StatusInternalServerError)
			return
		}

		// Add the client to the WireGuard configuration.
		log.Info().Msgf("Adding WireGuard peer: PublicKey=%s, ClientVPNIP=%s, ClientIP=%s\n", publicKey, clientVPNIP, clientIP)
		if err := im.AddPeer(im.GetInterfaceName(), publicKey, clientVPNIP, clientIP); err != nil {
			http.Error(w, "Failed to configure WireGuard tunnel: "+err.Error(), http.StatusInternalServerError)
			return
		}

		serverConfig, err := im.GetServerConfig()
		if err != nil {
			http.Error(w, "Failed to get server configuration: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := WGResponse{
			Message:      "WireGuard tunnel created successfully",
			ClientVPNIP:  clientVPNIP,
			ServerPubKey: serverConfig.PublicKey,
			ServerIP:     serverConfig.IP,
			ServerPort:   serverConfig.Port,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error().Err(err).Msg("Failed to encode response")
		}
	}
}
