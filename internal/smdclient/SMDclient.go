package smdclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	base "github.com/Cray-HPE/hms-base"
	"github.com/OpenCHAMI/smd/v2/pkg/sm"
	"github.com/rs/zerolog/log"
)

// Create an SMDClient Interface which can be more easily tested and mocked
type SMDClientInterface interface {
	IDfromMAC(mac string) (string, error)
	IDfromIP(ipaddr string) (string, error)
	IPfromID(id string) (string, error)
	MACfromID(id string) (string, error)
	GroupMembership(id string) ([]string, error)
	ComponentInformation(id string) (base.Component, error)
	PopulateNodes()
	ClusterName() string
	AddWGIP(id string, wgip string) error
	WGIPfromID(id string) (string, error)
}

// Add client usage examples
// unit testing
// golang lint
// Expand this client to handle more of the SMD API and work more directly with the resources it manages

var (
	ErrUnmarshal = errors.New("cannot unmarshal JSON")
)

// SMDClient is a client for SMD
type SMDClient struct {
	clusterName       string
	smdClient         *http.Client
	smdBaseURL        string
	tokenEndpoint     string
	accessToken       string
	nodes             map[string]NodeMapping
	nodesMutex        *sync.Mutex
	nodes_last_update time.Time
	stopCacheRefresh  chan struct{}
	stopOnce          sync.Once
}

type NodeInterface struct {
	MAC  string `json:"mac" yaml:"mac"`
	IP   string `json:"ip" yaml:"ip"`
	WGIP string `json:"wgip" yaml:"wgip"`
	Desc string `json:"description" yaml:"description"`
}

type NodeMapping struct {
	Xname      string          `json:"xname" yaml:"xname"`
	Interfaces []NodeInterface `json:"interfaces" yaml:"interfaces"`
}

// NewSMDClient creates a new SMDClient which connects to the SMD server at baseurl
// and uses the provided JWT server for authentication
func NewSMDClient(clusterName, baseurl, jwtURL, accessToken, certPath string, insecure bool) (*SMDClient, error) {
	var (
		c        *http.Client
		certPool *x509.CertPool
	)

	c = &http.Client{Timeout: 2 * time.Second}

	// try and load the cert if path is provided first
	if certPath != "" {
		cacert, err := os.ReadFile(certPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read cert from path %s: %v", certPath, err)
		}
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(cacert)
	}

	// set up the HTTP client's config
	c.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            certPool,
			InsecureSkipVerify: insecure,
		},
		DisableKeepAlives: true,
		Dial: (&net.Dialer{
			Timeout:   120 * time.Second,
			KeepAlive: 120 * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   120 * time.Second,
		ResponseHeaderTimeout: 120 * time.Second,
	}

	client := &SMDClient{
		clusterName:       clusterName,
		smdClient:         c,
		smdBaseURL:        baseurl,
		tokenEndpoint:     jwtURL,
		accessToken:       accessToken,
		nodesMutex:        &sync.Mutex{},
		nodes_last_update: time.Now(),
		nodes:             make(map[string]NodeMapping),
		stopCacheRefresh:  make(chan struct{}),
	}

	// Populate the cache initially
	client.PopulateNodes()

	// Start the cache refresh goroutine
	go client.startCacheRefresh()

	return client, nil
}

// startCacheRefresh starts a goroutine that refreshes the cache every minute
func (s *SMDClient) startCacheRefresh() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Debug().Msg("Ticker triggered. Refreshing cache")
			s.RefreshCache()
		case <-s.stopCacheRefresh:
			ticker.Stop()
			return
		}
	}
}

// RefreshCache refreshes the cache
func (s *SMDClient) RefreshCache() {
	log.Debug().Msg("Refreshing SMD cache")
	s.PopulateNodes()
}

// StopCacheRefresh stops the cache refresh goroutine
func (s *SMDClient) StopCacheRefresh() {
	s.stopOnce.Do(func() {
		close(s.stopCacheRefresh)
	})
	close(s.stopCacheRefresh)
}

// ClusterName returns the name of the cluster
func (s *SMDClient) ClusterName() string {
	return s.clusterName
}

// getSMD is a helper function to initialize the SMDClient
func (s *SMDClient) getSMD(ep string, smd interface{}) error {
	url := s.smdBaseURL + ep
	var resp *http.Response
	// Manage fetching a new JWT if we initially fail
	freshToken := false
	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+s.accessToken)
		resp, err = s.smdClient.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusUnauthorized {
			// Request failed; handle appropriately (based on whether or not
			// this was a fresh JWT)
			log.Info().Msg("Cached JWT was rejected by SMD")
			if !freshToken {
				log.Info().Msg("Fetching new JWT and retrying...")
				// Try to refresh the token and retry once
				if err2 := s.RefreshToken(); err2 != nil {
					// If token refresh fails, refresh will attempt again.
					// While effectively we could ignore the error, it helps
					// to see why the failure is occurring in case the error
					// is unusual (RefreshToken() has a few different failure
					// modes).
					log.Debug().Err(err).Msg("failed to refresh token")
				}
				freshToken = true
			} else {
				log.Info().Msg("SMD authentication failed, even with a fresh" +
					" JWT. Something has gone terribly wrong; exiting to" +
					" avoid invalid request spam.")
				os.Exit(2)
			}
		} else {
			// Request succeeded; we're done here
			break
		}
	}
	defer func() {
		_ = resp.Body.Close() // ignoring error on deferred Close
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("failed to read response body")
		return err
	}
	if err := json.Unmarshal(body, smd); err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("failed to unmarshal SMD response")
		return ErrUnmarshal
	}
	return nil
}

// PopulateNodes fetches the Ethernet interface data from the SMD server and populates the nodes map
// with the corresponding node information, including MAC addresses, IP addresses, and descriptions.
func (s *SMDClient) PopulateNodes() {
	s.nodesMutex.Lock()
	defer s.nodesMutex.Unlock()
	var ethIfaceArray []sm.CompEthInterfaceV2
	ep := "/hsm/v2/Inventory/EthernetInterfaces/"
	if err := s.getSMD(ep, &ethIfaceArray); err != nil {
		log.Error().Err(err).Msg("Failed to get SMD data")
		return
	}
	log.Debug().Msgf("Populating nodes with %d Ethernet interfaces", len(ethIfaceArray))
	for _, ep := range ethIfaceArray {
		if existingNode, exists := s.nodes[ep.CompID]; exists {
			found := false
			for index, existingInterface := range existingNode.Interfaces {
				if strings.EqualFold(existingInterface.MAC, ep.MACAddr) {
					// found the interface.  Update the IP and Description
					found = true
					// Update the IP and Description
					if len(ep.IPAddrs) > 0 {
						existingInterface.IP = ep.IPAddrs[0].IPAddr
					}
					existingInterface.Desc = ep.Desc
					existingNode.Interfaces[index] = existingInterface
				}
			}
			if !found {
				// This is a new interface.  Add it to the map
				newInterface := NodeInterface{
					MAC:  ep.MACAddr,
					Desc: ep.Desc,
				}
				if len(ep.IPAddrs) > 0 {
					newInterface.IP = ep.IPAddrs[0].IPAddr
				}
				existingNode.Interfaces = append(existingNode.Interfaces, newInterface)
				s.nodes[ep.CompID] = existingNode
			}
		} else { // This is a new node
			newNode := NodeMapping{
				Xname: ep.CompID,
			}
			newInterface := NodeInterface{
				MAC:  ep.MACAddr,
				Desc: ep.Desc,
			}
			log.Debug().Msgf("Adding new node %s with MAC %s and IPs: %v", ep.CompID, ep.MACAddr, ep.IPAddrs)
			if len(ep.IPAddrs) > 0 {
				newInterface.IP = ep.IPAddrs[0].IPAddr
			}
			newNode.Interfaces = append(newNode.Interfaces, newInterface)
			s.nodes[ep.CompID] = newNode
		}
	}
	s.nodes_last_update = time.Now()
	log.Debug().Msg("Nodes map populated")
}

// IDfromMAC returns the ID of the xname that has the MAC address
func (s *SMDClient) IDfromMAC(mac string) (string, error) {
	s.nodesMutex.Lock()
	defer s.nodesMutex.Unlock()

	for _, node := range s.nodes {
		for _, iface := range node.Interfaces {
			if strings.EqualFold(mac, iface.MAC) {
				return node.Xname, nil
			}
		}
	}
	return "", errors.New("MAC " + mac + " not found for an xname in nodes")
}

// IDfromIP returns the ID of the xname that has the IP address
func (s *SMDClient) IDfromIP(ipaddr string) (string, error) {
	s.nodesMutex.Lock()
	defer s.nodesMutex.Unlock()

	for _, node := range s.nodes {
		for _, iface := range node.Interfaces {
			if strings.EqualFold(ipaddr, iface.IP) || strings.EqualFold(ipaddr, iface.WGIP) {
				return node.Xname, nil
			}
		}
	}
	return "", errors.New("IP address " + ipaddr + " not found for an xname in nodes")
}

// IPfromID returns the IP address of the xname with the given ID
func (s *SMDClient) IPfromID(id string) (string, error) {
	s.nodesMutex.Lock()
	defer s.nodesMutex.Unlock()
	if node, found := s.nodes[id]; found {
		if node.Interfaces != nil {
			if len(node.Interfaces) > 0 {
				return node.Interfaces[0].IP, nil
			}
			return "", errors.New("no interfaces found for ID " + id)
		}
	}
	return "", errors.New("ID " + id + " not found in nodes")
}

func (s *SMDClient) MACfromID(id string) (string, error) {
	s.nodesMutex.Lock()
	defer s.nodesMutex.Unlock()
	if node, found := s.nodes[id]; found {
		if node.Interfaces != nil {
			if len(node.Interfaces) > 0 {
				return node.Interfaces[0].MAC, nil
			}
			return "", errors.New("no interfaces found for ID " + id)
		}
	}
	return "", errors.New("ID " + id + " not found in nodes")
}

// GroupMembership returns the group labels for the xname with the given ID
func (s *SMDClient) GroupMembership(id string) ([]string, error) {
	if id == "" {
		log.Err(errors.New("ID is empty")).Msg("ID is empty")
	}
	ml := new(sm.Membership)
	ep := "/hsm/v2/memberships/" + id
	err := s.getSMD(ep, ml)
	if err != nil {
		return nil, err
	}
	return ml.GroupLabels, nil
}

func (s *SMDClient) ComponentInformation(id string) (base.Component, error) {
	var node base.Component
	ep := "/hsm/v2/State/Components/" + id
	err := s.getSMD(ep, &node)
	if err != nil {
		return node, err
	}
	return node, nil
}

func (s *SMDClient) AddWGIP(id string, wgip string) error {
	s.nodesMutex.Lock()
	defer s.nodesMutex.Unlock()
	if node, found := s.nodes[id]; found {
		if node.Interfaces != nil {
			if len(node.Interfaces) > 0 {
				node.Interfaces[0].WGIP = wgip
				return nil
			}
			return errors.New("no interfaces found for ID " + id)
		}
	}
	return nil
}

func (s *SMDClient) WGIPfromID(id string) (string, error) {
	s.nodesMutex.Lock()
	defer s.nodesMutex.Unlock()
	if node, found := s.nodes[id]; found {
		if node.Interfaces != nil {
			if len(node.Interfaces) > 0 {
				return node.Interfaces[0].WGIP, nil
			}
			return "", errors.New("no interfaces found for ID " + id)
		}
	}
	return "", errors.New("ID " + id + " not found in nodes")
}
