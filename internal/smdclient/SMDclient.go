package smdclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	base "github.com/Cray-HPE/hms-base"
	"github.com/OpenCHAMI/smd/v2/pkg/sm"
	"github.com/rs/zerolog/log"
)

// Create an SMDClient Interface which can be more easily tested and mocked
type SMDClientInterface interface {
	IDfromMAC(mac string) (string, error)
	IDfromIP(ipaddr string) (string, error)
	GroupMembership(id string) ([]string, error)
	ComponentInformation(id string) (base.Component, error)
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
	smdClient     *http.Client
	smdBaseURL    string
	tokenEndpoint string
	accessToken   string
}

// NewSMDClient creates a new SMDClient which connects to the SMD server at baseurl
// and uses the provided JWT server for authentication
func NewSMDClient(baseurl string, jwtURL string, accessToken string, certPath string, insecure bool) *SMDClient {
	c := &http.Client{Timeout: 2 * time.Second}
	if insecure {
		c.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	} else {
		cacert, err := os.ReadFile(certPath)
		if err != nil {
			log.Error().Err(err).Msgf("failed to read cert from path %s", certPath)
			return nil
		}
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(cacert)

		// add cert pool to client if valid
		if certPool != nil {
			// make sure that we can access the internal client
			c.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:            certPool,
					InsecureSkipVerify: true,
				},
				DisableKeepAlives: true,
				Dial: (&net.Dialer{
					Timeout:   120 * time.Second,
					KeepAlive: 120 * time.Second,
				}).Dial,
				TLSHandshakeTimeout:   120 * time.Second,
				ResponseHeaderTimeout: 120 * time.Second,
			}
		}

	}
	return &SMDClient{
		smdClient:     c,
		smdBaseURL:    baseurl,
		tokenEndpoint: jwtURL,
		accessToken:   accessToken,
	}
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
				s.RefreshToken()
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
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, smd); err != nil {
		return ErrUnmarshal
	}
	return nil
}

// IDfromMAC returns the ID of the xname that has the MAC address
func (s *SMDClient) IDfromMAC(mac string) (string, error) {
	var ethIfaceArray []sm.CompEthInterfaceV2
	ep := "/hsm/v2/Inventory/EthernetInterfaces/"
	s.getSMD(ep, &ethIfaceArray)

	for _, ep := range ethIfaceArray {
		if strings.EqualFold(mac, ep.MACAddr) {
			return ep.CompID, nil
		}
	}
	return "", errors.New("MAC " + mac + " not found for an xname in EthernetInterfaces")
}

// IDfromIP returns the ID of the xname that has the IP address
func (s *SMDClient) IDfromIP(ipaddr string) (string, error) {
	var ethIfaceArray []sm.CompEthInterfaceV2
	ep := "/hsm/v2/Inventory/EthernetInterfaces/"
	s.getSMD(ep, &ethIfaceArray)

	for _, ep := range ethIfaceArray {
		for _, v := range ep.IPAddrs {
			if strings.EqualFold(ipaddr, v.IPAddr) {
				return ep.CompID, nil
			}
		}
	}
	return "", errors.New("IP address " + ipaddr + " not found for an xname in EthernetInterfaces")
}

// GroupMembership returns the group labels for the xname with the given ID
func (s *SMDClient) GroupMembership(id string) ([]string, error) {
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
