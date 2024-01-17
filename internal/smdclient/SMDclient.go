package smdclient

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/OpenCHAMI/smd/v2/pkg/sm"
)

// Add client usage examples
// unit testing
// golang lint
// Expand this client to handle more of the SMD API and work more directly with the resources it manages

var (
	UnMarsharlError = errors.New("Can not unmarshal JSON")
)

// godoc ?
// SMDClient is a client for SMD
type SMDClient struct {
	smdClient  *http.Client
	smdBaseURL string
}

// NewSMDClient creates a new SMDClient which connects to the SMD server at baseurl
func NewSMDClient(baseurl string) *SMDClient {
	c := &http.Client{Timeout: 2 * time.Second}
	return &SMDClient{
		smdClient:  c,
		smdBaseURL: baseurl,
	}
}

// getSMD is a helper function to initialize the SMDClient
func (s *SMDClient) getSMD(ep string, smd interface{}) error {
	url := s.smdBaseURL + ep
	resp, err := s.smdClient.Get(url)
	if err != nil {
		return err
	}
	// check http retrun value
	defer resp.Body.Close()
	// ioutil is deprecated
	body, err := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, smd); err != nil {
		return UnMarsharlError
	}
	return nil
}

// IDfromMAC returns the ID of the xname that has the MAC address
func (s *SMDClient) IDfromMAC(mac string) (string, error) {
	endpointData := new(sm.ComponentEndpointArray)
	ep := "/hsm/v2/Inventory/ComponentEndpoints/"
	s.getSMD(ep, endpointData)

	for _, ep := range endpointData.ComponentEndpoints {
		id := ep.ID
		nics := ep.RedfishSystemInfo.EthNICInfo
		for _, v := range nics {
			if strings.EqualFold(mac, v.MACAddress) {
				return id, nil
			}
		}
	}
	return "", errors.New("MAC " + mac + " not found for an xname in CompenentEndpoints")
}

// GroupMembership returns the group labels for the xname with the given ID
func (s *SMDClient) GroupMembership(id string) ([]string, error) {
	ml := new(sm.Membership)
	ep := "/hsm/v2/memberships/" + id
	s.getSMD(ep, ml)
	return ml.GroupLabels, nil
}
