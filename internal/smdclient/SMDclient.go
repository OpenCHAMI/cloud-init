// SPDX-FileCopyrightText: 2026 Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

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
	"github.com/openchami/smd/v2/pkg/sm"
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
	ComponentInformationWithRetry(id string, maxRetries int) (base.Component, error)
	PopulateNodes()
	ClusterName() string
	AddWGIP(id string, wgip string) error
	WGIPfromID(id string) (string, error)
}

// Add client usage examples
// unit testing
// golang lint
// Expand this client to handle more of the SMD API and work more directly with the resources it manages

// SMDClient is a client for SMD
type SMDClient struct {
	clusterName       string
	smdClient         *http.Client
	smdBaseURL        string
	tokenEndpoint     string
	accessToken       string
	accessTokenMutex  sync.Mutex
	nodes             map[string]NodeMapping
	components        map[string]base.Component
	nodesMutex        *sync.RWMutex
	nodes_last_update time.Time
	stopCacheRefresh  chan struct{}
	stopOnce          sync.Once
	// Reverse indexes for O(1) lookups
	ipToXname   map[string]string
	macToXname  map[string]string
	wgipToXname map[string]string
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
	Groups     []string        `json:"groups" yaml:"groups"`
}

// NewSMDClient creates a new SMDClient which connects to the SMD server at baseurl
// and uses the provided JWT server for authentication
func NewSMDClient(clusterName, baseurl, jwtURL, accessToken, certPath string, insecure bool) (*SMDClient, error) {
	var (
		c        *http.Client
		certPool *x509.CertPool
	)

	c = &http.Client{Timeout: 10 * time.Second}

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
		DisableKeepAlives: false,
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
		nodesMutex:        &sync.RWMutex{},
		nodes_last_update: time.Now(),
		nodes:             make(map[string]NodeMapping),
		components:        make(map[string]base.Component),
		stopCacheRefresh:  make(chan struct{}),
		ipToXname:         make(map[string]string),
		macToXname:        make(map[string]string),
		wgipToXname:       make(map[string]string),
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
func (s *SMDClient) getSMD(ep string, smd any) error {
	url := s.smdBaseURL + ep
	var resp *http.Response
	// Manage fetching a new JWT if we initially fail
	freshToken := false
	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		usedToken := s.currentAccessToken()
		req.Header.Set("Authorization", "Bearer "+usedToken)
		resp, err = s.smdClient.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusUnauthorized {
			_ = resp.Body.Close()
			// Request failed; handle appropriately (based on whether or not
			// this was a fresh JWT)
			log.Info().Msg("Cached JWT was rejected by SMD")
			if !freshToken {
				log.Info().Msg("Fetching new JWT and retrying...")
				// Try to refresh the token and retry once
				if err2 := s.refreshTokenIfCurrent(usedToken); err2 != nil {
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
	if resp.StatusCode >= 400 {
		log.Error().Msgf("SMD GET request went through, but returned unsuccessful HTTP response (HTTP %d)", resp.StatusCode)
		return ErrSMDResponse{HTTPResponse: resp}
	}
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

func (s *SMDClient) currentAccessToken() string {
	s.accessTokenMutex.Lock()
	defer s.accessTokenMutex.Unlock()
	return s.accessToken
}

// PopulateNodes fetches the Ethernet interface data from the SMD server and populates the nodes map
// with the corresponding node information, including MAC addresses, IP addresses, descriptions, and group membership.
func (s *SMDClient) PopulateNodes() {
	var ethIfaceArray []sm.CompEthInterfaceV2
	ep := "/hsm/v2/Inventory/EthernetInterfaces/"
	if err := s.getSMD(ep, &ethIfaceArray); err != nil {
		log.Error().Err(err).Msg("Failed to get SMD data")
		return
	}

	nextNodes := make(map[string]NodeMapping)
	log.Debug().Msgf("Populating nodes with %d Ethernet interfaces", len(ethIfaceArray))
	for _, ethIface := range ethIfaceArray {
		if existingNode, exists := nextNodes[ethIface.CompID]; exists {
			found := false
			for index, existingInterface := range existingNode.Interfaces {
				if strings.EqualFold(existingInterface.MAC, ethIface.MACAddr) {
					// found the interface.  Update the IP and Description
					found = true
					// Update the IP and Description
					if len(ethIface.IPAddrs) > 0 {
						existingInterface.IP = ethIface.IPAddrs[0].IPAddr
					}
					existingInterface.Desc = ethIface.Desc
					existingNode.Interfaces[index] = existingInterface
				}
			}
			if !found {
				// This is a new interface.  Add it to the map
				newInterface := NodeInterface{
					MAC:  ethIface.MACAddr,
					Desc: ethIface.Desc,
				}
				if len(ethIface.IPAddrs) > 0 {
					newInterface.IP = ethIface.IPAddrs[0].IPAddr
				}
				existingNode.Interfaces = append(existingNode.Interfaces, newInterface)
			}
			nextNodes[ethIface.CompID] = existingNode
		} else { // This is a new node
			newNode := NodeMapping{
				Xname: ethIface.CompID,
			}
			newInterface := NodeInterface{
				MAC:  ethIface.MACAddr,
				Desc: ethIface.Desc,
			}
			log.Debug().Msgf("Adding new node %s with MAC %s and IPs: %v", ethIface.CompID, ethIface.MACAddr, ethIface.IPAddrs)
			if len(ethIface.IPAddrs) > 0 {
				newInterface.IP = ethIface.IPAddrs[0].IPAddr
			}
			newNode.Interfaces = append(newNode.Interfaces, newInterface)
			nextNodes[ethIface.CompID] = newNode
		}
	}

	var componentArray base.ComponentArray
	if err := s.getSMD("/hsm/v2/State/Components", &componentArray); err != nil {
		log.Error().Err(err).Msg("Failed to get SMD component data")
		return
	}
	nextComponents := make(map[string]base.Component, len(componentArray.Components))
	for _, component := range componentArray.Components {
		if component == nil || component.ID == "" {
			continue
		}
		nextComponents[component.ID] = cloneComponent(*component)
	}
	if len(nextComponents) == 0 {
		log.Error().Msg("SMD component data was empty")
		return
	}
	for xname := range nextNodes {
		if _, found := nextComponents[xname]; !found {
			log.Error().Str("xname", xname).Msg("SMD component data missing node from Ethernet interface inventory")
			return
		}
	}

	log.Debug().Msg("Fetching group membership for all nodes")
	memberships := make([]sm.Membership, 0)
	if err := s.getSMD("/hsm/v2/memberships?type=node", &memberships); err != nil {
		log.Error().Err(err).Msg("Failed to get SMD node memberships")
		return
	}

	groupsByXname := make(map[string][]string, len(memberships))
	for _, membership := range memberships {
		groupsByXname[membership.ID] = membership.GroupLabels
	}
	for xname, node := range nextNodes {
		node.Groups = []string{}
		if groups, found := groupsByXname[xname]; found && groups != nil {
			node.Groups = groups
		}
		nextNodes[xname] = node
	}

	// Build reverse indexes for O(1) lookups
	log.Debug().Msg("Building reverse indexes")
	nextIPToXname := make(map[string]string)
	nextMACToXname := make(map[string]string)
	nextWGIPToXname := make(map[string]string)

	for xname, node := range nextNodes {
		for _, iface := range node.Interfaces {
			if iface.IP != "" {
				nextIPToXname[strings.ToLower(iface.IP)] = xname
			}
			if iface.MAC != "" {
				nextMACToXname[strings.ToLower(iface.MAC)] = xname
			}
			if iface.WGIP != "" {
				nextWGIPToXname[strings.ToLower(iface.WGIP)] = xname
			}
		}
	}

	s.nodesMutex.Lock()
	defer s.nodesMutex.Unlock()

	for xname, nextNode := range nextNodes {
		currentNode, found := s.nodes[xname]
		if !found {
			continue
		}
		for nextIndex, nextInterface := range nextNode.Interfaces {
			for _, currentInterface := range currentNode.Interfaces {
				if currentInterface.WGIP == "" || !strings.EqualFold(currentInterface.MAC, nextInterface.MAC) {
					continue
				}
				nextNode.Interfaces[nextIndex].WGIP = currentInterface.WGIP
				nextWGIPToXname[strings.ToLower(currentInterface.WGIP)] = xname
				break
			}
		}
		nextNodes[xname] = nextNode
	}

	s.nodes = nextNodes
	s.components = nextComponents
	s.ipToXname = nextIPToXname
	s.macToXname = nextMACToXname
	s.wgipToXname = nextWGIPToXname
	s.nodes_last_update = time.Now()
	log.Debug().Msgf("Nodes map populated with %d nodes, %d IP mappings, %d MAC mappings",
		len(nextNodes), len(nextIPToXname), len(nextMACToXname))
}

// IDfromMAC returns the ID of the xname that has the MAC address
func (s *SMDClient) IDfromMAC(mac string) (string, error) {
	s.nodesMutex.RLock()
	defer s.nodesMutex.RUnlock()

	key := strings.ToLower(mac)
	if xname, found := s.macToXname[key]; found {
		return xname, nil
	}
	return "", fmt.Errorf("MAC %s not found for an xname in nodes", mac)
}

// IDfromIP returns the ID of the xname that has the IP address
func (s *SMDClient) IDfromIP(ipaddr string) (string, error) {
	s.nodesMutex.RLock()
	defer s.nodesMutex.RUnlock()

	key := strings.ToLower(ipaddr)
	if xname, found := s.ipToXname[key]; found {
		return xname, nil
	}
	if xname, found := s.wgipToXname[key]; found {
		return xname, nil
	}
	return "", fmt.Errorf("IP address %s not found for an xname in nodes", ipaddr)
}

// IPfromID returns the IP address of the xname with the given ID
func (s *SMDClient) IPfromID(id string) (string, error) {
	s.nodesMutex.RLock()
	defer s.nodesMutex.RUnlock()
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
	s.nodesMutex.RLock()
	defer s.nodesMutex.RUnlock()
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
		err := errors.New("ID is empty")
		log.Err(err).Msg("failed to get group membership")
		return []string{}, err
	}

	s.nodesMutex.RLock()
	defer s.nodesMutex.RUnlock()

	if node, found := s.nodes[id]; found {
		return node.Groups, nil
	}

	return []string{}, fmt.Errorf("node %s not found in cache", id)
}

func (s *SMDClient) ComponentInformation(id string) (base.Component, error) {
	var node base.Component
	if strings.Trim(id, " \t") == "" {
		return node, ErrEmptyID
	}

	s.nodesMutex.RLock()
	if component, found := s.components[id]; found {
		s.nodesMutex.RUnlock()
		return cloneComponent(component), nil
	}
	s.nodesMutex.RUnlock()

	ep := "/hsm/v2/State/Components/" + id
	err := s.getSMD(ep, &node)
	if err != nil {
		return node, err
	}
	return node, nil
}

func cloneComponent(component base.Component) base.Component {
	if component.Enabled != nil {
		enabled := *component.Enabled
		component.Enabled = &enabled
	}
	return component
}

// ComponentInformationWithRetry wraps ComponentInformation with exponential backoff retry logic
// for transient network errors and timeouts. This is critical during boot storms when SMD
// may be temporarily overloaded.
func (s *SMDClient) ComponentInformationWithRetry(id string, maxRetries int) (base.Component, error) {
	var lastErr error
	var node base.Component

	for attempt := range maxRetries {
		node, err := s.ComponentInformation(id)
		if err == nil {
			// Success - return immediately
			return node, nil
		}

		lastErr = err

		// Check if error is retryable (timeout or network error)
		if !isRetryableError(err) {
			// Non-retryable error (e.g., 404 Not Found, auth failure)
			return node, err
		}

		// Don't sleep on the last attempt
		if attempt < maxRetries-1 {
			// Exponential backoff: 100ms, 200ms, 400ms, 800ms, etc.
			backoff := time.Duration(100<<uint(attempt)) * time.Millisecond
			log.Warn().
				Str("component_id", id).
				Int("attempt", attempt+1).
				Int("max_retries", maxRetries).
				Dur("backoff", backoff).
				Err(err).
				Msg("SMD request failed, retrying after backoff")
			time.Sleep(backoff)
		}
	}

	// All retries exhausted
	log.Error().
		Str("component_id", id).
		Int("attempts", maxRetries).
		Err(lastErr).
		Msg("SMD request failed after all retry attempts")

	return node, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Timeout errors
	if strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "Client.Timeout exceeded") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "Timeout") {
		return true
	}

	// Network errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "no route to host") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "EOF") {
		return true
	}

	// Temporary service unavailability (503)
	if strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "Service Unavailable") {
		return true
	}

	// Too Many Requests (429) - SMD may be rate limiting
	if strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "Too Many Requests") {
		return true
	}

	return false
}

func (s *SMDClient) AddWGIP(id string, wgip string) error {
	s.nodesMutex.Lock()
	defer s.nodesMutex.Unlock()
	if node, found := s.nodes[id]; found {
		if node.Interfaces != nil {
			if len(node.Interfaces) > 0 {
				node.Interfaces[0].WGIP = wgip
				s.nodes[id] = node
				// Update reverse index
				s.wgipToXname[strings.ToLower(wgip)] = id
				return nil
			}
			return errors.New("no interfaces found for ID " + id)
		}
	}
	return nil
}

func (s *SMDClient) WGIPfromID(id string) (string, error) {
	s.nodesMutex.RLock()
	defer s.nodesMutex.RUnlock()
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
