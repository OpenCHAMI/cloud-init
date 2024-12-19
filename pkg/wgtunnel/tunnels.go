package wgtunnel

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

type PeerConfig struct {
	PublicKey string     `json:"public_key"`
	IP        net.IPAddr `json:"ip"`
}

type ServerConfig struct {
	PublicKey string `json:"public_key"`
	IP        string `json:"ip"`
	Port      string `json:"port"`
}

type Store interface {
	IpForPeer(peerName, publicKey string) string
	GetInterfaceName() string
	GetServerConfig() (ServerConfig, error)
}

type InterfaceManager struct {
	listenPort      int
	interfaceName   string
	network         net.IPNet
	peers           map[string]PeerConfig
	peersMutex      sync.RWMutex
	allocatedIPs    []net.IPAddr
	lastallocatedIP *net.IPAddr
	privateKey      string
}

func (m *InterfaceManager) GetServerConfig() (ServerConfig, error) {
	return ServerConfig{
		PublicKey: m.privateKey,
		IP:        m.network.IP.String(),
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
		allocatedIPs:  make([]net.IPAddr, 0),
	}
	im.privateKey, err = generateKey()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to generate private key")
	}
	return &im
}

func (m *InterfaceManager) IpForPeer(peerName string, publicKey string) string {
	m.peersMutex.RLock()
	defer m.peersMutex.RUnlock()
	if _, ok := m.peers[peerName]; !ok {
		// Peer not found.  Store the peer and return the IP.
		ip := m.allocateIP(peerName)
		m.peers[peerName] = PeerConfig{
			IP:        ip,
			PublicKey: publicKey,
		}
	}
	return m.peers[peerName].IP.IP.String()
}

func (m *InterfaceManager) allocateIP(peerName string) net.IPAddr {
	m.peersMutex.Lock()
	defer m.peersMutex.Unlock()
	ip := m.nextIP()
	m.peers[peerName] = PeerConfig{
		IP: ip,
	}
	return ip
}

func (m *InterfaceManager) nextIP() net.IPAddr {
	m.peersMutex.Lock()
	defer m.peersMutex.Unlock()
	if m.lastallocatedIP == nil {
		m.lastallocatedIP = &net.IPAddr{
			IP: m.network.IP,
		}
	} else {
		m.lastallocatedIP.IP = m.lastallocatedIP.IP.To4()
		m.lastallocatedIP.IP[3]++
	}
	return *m.lastallocatedIP
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
	cmd := exec.Command("wg", "pubkey")
	cmd.Stdin = strings.NewReader(m.privateKey)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to generate public key: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (m *InterfaceManager) StartServer() error {
	// Step 1: Create the WireGuard interface
	if err := exec.Command("ip", "link", "add", "dev", m.interfaceName, "type", "wireguard").Run(); err != nil {
		if !strings.Contains(err.Error(), "File exists") { // Skip if interface already exists
			return fmt.Errorf("failed to create WireGuard interface: %v", err)
		}
	}

	// Step 3: Assign IP address to the WireGuard interface
	if err := exec.Command("ip", "address", "add", "dev", m.interfaceName, m.network.String()).Run(); err != nil {
		return fmt.Errorf("failed to assign IP address to interface: %v", err)
	}

	// Step 4: Set the private key and listen port
	cmd := exec.Command("sh", "-c", fmt.Sprintf("wg set %s listen-port %d private-key <(echo %s)", m.interfaceName, m.listenPort, m.privateKey))
	cmd.Stdin = strings.NewReader(m.privateKey)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure WireGuard: %v", err)
	}

	// Step 5: Bring the interface up
	if err := exec.Command("ip", "link", "set", "up", "dev", m.interfaceName).Run(); err != nil {
		return fmt.Errorf("failed to bring up the WireGuard interface: %v", err)
	}

	log.Printf("WireGuard server configured: Interface=%s, Port=%d\n", m.interfaceName, m.listenPort)
	return nil

}

// AddWireGuardPeer adds a peer to the WireGuard configuration.
func AddWireGuardPeer(interfaceID, publicKey, vpnIP, clientIP string) error {
	allowedIPs := vpnIP + "/32"

	cmd := exec.Command("wg", "set", interfaceID,
		"peer", publicKey,
		"allowed-ips", allowedIPs,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add WireGuard peer: %v", err)
	}

	// Update routing if necessary.
	cmd = exec.Command("ip", "route", "add", clientIP, "via", vpnIP, "dev", interfaceID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update routing: %v", err)
	}

	log.Info().
		Str("Public Key", publicKey).
		Str("Client IP", clientIP).
		Str("VPN IP", vpnIP).
		Msgf("Peer added: PublicKey=%s, VPNIP=%s, ClientIP=%s\n", publicKey, vpnIP, clientIP)
	return nil
}

// setupWireGuardServer sets up the initial WireGuard interface and configuration.
func SetupWireGuardServer(interfaceID string, listenPort int, ipmask string) error {
	// Step 1: Generate private/public keys if not already generated
	privateKey, err := generateKey()
	if err != nil {
		return fmt.Errorf("failed to generate private key: %v", err)
	}

	// Step 2: Create the WireGuard interface
	if err := exec.Command("ip", "link", "add", "dev", interfaceID, "type", "wireguard").Run(); err != nil {
		if !strings.Contains(err.Error(), "File exists") { // Skip if interface already exists
			return fmt.Errorf("failed to create WireGuard interface: %v", err)
		}
	}

	// Step 3: Assign IP address to the WireGuard interface
	if err := exec.Command("ip", "address", "add", "dev", interfaceID, ipmask).Run(); err != nil {
		return fmt.Errorf("failed to assign IP address to interface: %v", err)
	}

	// Step 4: Set the private key and listen port
	cmd := exec.Command("sh", "-c", fmt.Sprintf("wg set %s listen-port %d private-key <(echo %s)", interfaceID, listenPort, privateKey))
	cmd.Stdin = strings.NewReader(privateKey)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure WireGuard: %v", err)
	}

	// Step 5: Bring the interface up
	if err := exec.Command("ip", "link", "set", "up", "dev", interfaceID).Run(); err != nil {
		return fmt.Errorf("failed to bring up the WireGuard interface: %v", err)
	}

	log.Printf("WireGuard server configured: Interface=%s, Port=%d\n", interfaceID, listenPort)
	return nil
}

// generateKey generates a WireGuard private or public key.
func generateKey() (string, error) {
	cmd := exec.Command("wg", "genkey")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to generate key: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}
