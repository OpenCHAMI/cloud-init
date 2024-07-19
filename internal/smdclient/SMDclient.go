package smdclient

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/OpenCHAMI/smd/v2/pkg/sm"
)

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
func NewSMDClient(baseurl string, jwtURL string) *SMDClient {
	c := &http.Client{Timeout: 2 * time.Second}
	return &SMDClient{
		smdClient:   c,
		smdBaseURL:  baseurl,
		tokenEndpoint: jwtURL,
		accessToken: "",
	}
}

// getSMD is a helper function to initialize the SMDClient
func (s *SMDClient) getSMD(ep string, smd interface{}) error {
	url := s.smdBaseURL + ep
	var resp *http.Response
	// Manage fetching a new JWT if we initially fail
	freshToken := false
	for true {
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
			log.Println("Cached JWT was rejected by SMD")
			if !freshToken {
				log.Println("Fetching new JWT and retrying...")
				s.RefreshToken()
				freshToken = true
			} else {
				log.Fatalln("SMD authentication failed, even with a fresh" +
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

// Helper array type
// TODO: This probably belongs in OpenCHAMI/smd, pkg/sm/endpoints.go
type CompEthInterfaceV2Array struct {
	Array []*sm.CompEthInterfaceV2
}

// IDfromMAC returns the ID of the xname that has the MAC address
func (s *SMDClient) IDfromMAC(mac string) (string, error) {
	endpointData := new(CompEthInterfaceV2Array)
	ep := "/hsm/v2/Inventory/EthernetInterfaces/"
	s.getSMD(ep, endpointData)

	for _, ep := range endpointData.Array {
		if strings.EqualFold(mac, ep.MACAddr) {
			return ep.CompID, nil
		}
	}
	return "", errors.New("MAC " + mac + " not found for an xname in EthernetInterfaces")
}

// GroupMembership returns the group labels for the xname with the given ID
func (s *SMDClient) GroupMembership(id string) ([]string, error) {
	ml := new(sm.Membership)
	ep := "/hsm/v2/memberships/" + id
	s.getSMD(ep, ml)
	return ml.GroupLabels, nil
}
