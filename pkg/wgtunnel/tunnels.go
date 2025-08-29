package wgtunnel

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

type PeerConfig struct {
	PublicKey string     `json:"public_key" yaml:"public_key"`
	IP        net.IPAddr `json:"ip" yaml:"ip"`
}

type ServerConfig struct {
	PublicKey string `json:"public_key" yaml:"public_key"`
	IP        string `json:"ip" yaml:"ip"`
	Port      string `json:"port" yaml:"port"`
}

type Store interface {
	IpForPeer(peerName, publicKey string) string
	GetInterfaceName() string
	GetServerConfig() (ServerConfig, error)
}

type InterfaceManager struct {
	listenPort    int
	interfaceName string
	network       net.IPNet
	ipAddress     net.IPAddr
	peers         map[string]PeerConfig
	peersMutex    sync.RWMutex
	ipManager     *IPAllocator
	privateKey    string
	publicKey     string
}

func (m *InterfaceManager) GetServerConfig() (ServerConfig, error) {
	return ServerConfig{
		PublicKey: m.publicKey,
		IP:        m.ipAddress.String(),
		Port:      fmt.Sprintf("%d", m.listenPort),
	}, nil
}

func (m *InterfaceManager) GetInterfaceName() string {
	return m.interfaceName
}

func NewInterfaceManager(name string, localIp net.IP, network *net.IPNet) *InterfaceManager {
	var err error
	im := InterfaceManager{
		interfaceName: name,
		peers:         make(map[string]PeerConfig),
		peersMutex:    sync.RWMutex{},
		network:       *network,

		listenPort: 58036,
	}
	im.ipManager, err = NewIPAllocator(network.String())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create IP allocator")
	}
	im.privateKey, err = generateKey()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to generate private key")
	}
	wgIp, err := GetUsableIP(network)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get usable IP")
	}
	im.ipAddress = net.IPAddr{IP: wgIp, Zone: ""}
	_ = im.ipManager.Reserve(im.ipAddress)

	return &im
}

// GetUsableIP checks if the given IP in a net.IPNet is usable. If not, it returns the first usable IP in the subnet.
func GetUsableIP(network *net.IPNet) (net.IP, error) {
	// Ensure the IP is IPv4 (for simplicity, extend this for IPv6 if needed)
	ip := network.IP.To4()
	if ip == nil {
		return nil, errors.New("only IPv4 is supported")
	}

	// Calculate network and broadcast addresses
	networkAddr := network.IP.Mask(network.Mask)
	broadcastAddr := make(net.IP, len(networkAddr))
	for i := range networkAddr {
		broadcastAddr[i] = networkAddr[i] | ^network.Mask[i]
	}

	// Check if the given IP is usable
	if !ip.Equal(networkAddr) && !ip.Equal(broadcastAddr) {
		return ip, nil
	}

	// Return the first usable IP (network address + 1)
	firstUsableIP := make(net.IP, len(networkAddr))
	copy(firstUsableIP, networkAddr)
	firstUsableIP[3]++

	// Validate it falls within the subnet
	if !network.Contains(firstUsableIP) {
		return nil, errors.New("no usable IP in the subnet")
	}

	return firstUsableIP, nil
}

// IpForPeer allocates an IP address for a given peer based on its name and public key.
// If the peer already exists, it returns the existing IP address.
// Otherwise, it allocates a new IP address for the peer and stores the peer configuration.
func (m *InterfaceManager) IpForPeer(peerName string, publicKey string) string {
	m.peersMutex.RLock()
	defer m.peersMutex.RUnlock()
	log.Debug().Msgf("Allocating IP for peer: PeerName=%s, PublicKey=%s\n", peerName, publicKey)
	if _, ok := m.peers[peerName]; !ok {
		// Peer not found.  Store the peer and return the IP.
		ip, err := m.ipManager.NextAvailable()
		if err != nil {
			log.Error().Err(err).Msg("Failed to allocate IP address")
			return ""
		}

		m.peers[peerName] = PeerConfig{
			IP:        ip,
			PublicKey: publicKey,
		}
	} else { // Peer found.  Return the existing IP.
		log.Debug().Msgf("Peer already exists: PeerName=%s, PublicKey=%s\n", peerName, publicKey)
		m.peers[peerName] = PeerConfig{
			IP:        m.peers[peerName].IP,
			PublicKey: publicKey,
		}
	}
	log.Debug().Msgf("Allocated IP for peer: PeerName=%s, PublicKey=%s, IP=%s\n", peerName, publicKey, m.peers[peerName].IP.IP.String())
	return m.peers[peerName].IP.IP.String()
}

func (m *InterfaceManager) RemovePeer(peerName string) error {
	m.peersMutex.Lock()
	defer m.peersMutex.Unlock()
	if err := exec.Command("wg", "set", m.interfaceName, "peer", m.peers[peerName].PublicKey, "remove").Run(); err != nil {
		log.Error().Err(err).Msgf("Failed to remove peer (%s)", peerName)
		return err
	}
	delete(m.peers, peerName)
	return nil
}

func (m *InterfaceManager) GetPeers() map[string]PeerConfig {
	m.peersMutex.RLock()
	defer m.peersMutex.RUnlock()
	return m.peers
}

func (m *InterfaceManager) PublicKey() (string, error) {
	return m.publicKey, nil
}

func (m *InterfaceManager) StartServer() error {
	// Step 1: Create the WireGuard interface
	createInterfaceCommand := exec.Command("ip", "link", "add", "dev", m.interfaceName, "type", "wireguard")
	if out, err := createInterfaceCommand.CombinedOutput(); err != nil {
		if !strings.Contains(err.Error(), "File exists") { // Skip if interface already exists
			log.Warn().Str("output", string(out)).Msgf("Failed to assign IP address to interface: %v", err)
		}
	}

	// Step 2: Assign IP address to the WireGuard interface
	wgIp := m.ipAddress.IP.String()
	ones, _ := m.network.Mask.Size()
	wgCidr := fmt.Sprintf("%s/%d", wgIp, ones)

	if out, err := exec.Command("ip", "address", "add", "dev", m.interfaceName, wgCidr).CombinedOutput(); err != nil {
		log.Error().Str("output", string(out)).Msgf("Failed to assign IP address to interface: %v", err)
		return fmt.Errorf("failed to assign IP address to interface: %v", err)
	}

	// Step 3: Set the private key and listen port
	// Write the private key to a temporary file
	tmpfile, err := os.CreateTemp("", "wg-privatekey")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for private key: %v", err)
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	if _, err := tmpfile.WriteString(m.privateKey); err != nil {
		return fmt.Errorf("failed to write private key to temporary file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %v", err)
	}

	// Use the temporary file in the wg set command
	cmd := exec.Command("wg", "set", m.interfaceName, "listen-port", fmt.Sprintf("%d", m.listenPort), "private-key", tmpfile.Name())
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Error().Str("output", string(out)).Msgf("Failed to configure WireGuard: %v", err)
		return fmt.Errorf("failed to configure WireGuard: %v", err)
	}

	// Step 4: Bring the interface up
	if err := exec.Command("ip", "link", "set", "up", "dev", m.interfaceName).Run(); err != nil {
		return fmt.Errorf("failed to bring up the WireGuard interface: %v", err)
	}

	// Step 5: Store the public key
	cmd = exec.Command("wg", "show", m.interfaceName, "public-key")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get public key: %v", err)
	}
	m.publicKey = strings.TrimSpace(string(output))

	log.Info().
		Str("Interface Name", m.interfaceName).
		Str("Private Key", m.privateKey).
		Str("Public Key", m.publicKey).
		Int("Listen Port", m.listenPort).
		Str("IP Address", m.ipAddress.String()).
		Msg("Wireguard Server Configured")
	return nil

}

func (m *InterfaceManager) StopServer() error {
	// Step 1: Bring the interface down
	if err := exec.Command("ip", "link", "set", "down", "dev", m.interfaceName).Run(); err != nil {
		return fmt.Errorf("failed to bring down the WireGuard interface: %v", err)
	}

	// Step 2: Delete the WireGuard interface
	if err := exec.Command("ip", "link", "delete", "dev", m.interfaceName).Run(); err != nil {
		return fmt.Errorf("failed to delete the WireGuard interface: %v", err)
	}

	return nil
}

func (m *InterfaceManager) AddPeer(peerName, publicKey, vpnIP, clientIP string) error {
	m.peersMutex.RLock()
	defer m.peersMutex.RUnlock()

	// Add the peer to the WireGuard configuration
	if err := AddWireGuardPeer(m.interfaceName, publicKey, vpnIP, clientIP); err != nil {
		return err
	}
	m.peers[peerName] = PeerConfig{
		PublicKey: publicKey,
		IP:        net.IPAddr{IP: net.ParseIP(vpnIP), Zone: ""},
	}
	return nil
}

// AddWireGuardPeer adds a peer to the WireGuard configuration.
func AddWireGuardPeer(interfaceID, publicKey, vpnIP, clientIP string) error {
	allowedIPs := vpnIP + "/32"

	cmd := exec.Command("wg", "set", interfaceID,
		"peer", publicKey,
		"allowed-ips", allowedIPs,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Error().Str("output", string(out)).Msgf("Failed to add WireGuard peer: %v", err)
		return fmt.Errorf("failed to add WireGuard peer: %v", err)
	}

	// Update routing if necessary.
	// cmd = exec.Command("ip", "route", "add", clientIP, "via", vpnIP, "dev", interfaceID)
	// if out, err := cmd.CombinedOutput(); err != nil {
	// 	log.Error().Str("output", string(out)).Msgf("Failed to update routing: %v", err)
	// 	return fmt.Errorf("failed to update routing: %v", err)
	// }

	log.Info().
		Str("Public Key", publicKey).
		Str("Client IP", clientIP).
		Str("VPN IP", vpnIP).
		Msgf("Peer added: PublicKey=%s, VPNIP=%s, ClientIP=%s\n", publicKey, vpnIP, clientIP)
	return nil
}

// generateKey generates a WireGuard private or public key.
func generateKey() (string, error) {
	cmd := exec.Command("wg", "genkey")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to generate key: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}
