package smdclient

import (
	"net/http"
	"encoding/json"
	"time"
	"io"
	"errors"
	"strings"

	"github.com/OpenCHAMI/smd/v2/pkg/sm"
)

// Add client usage examples
// unit testing
// golang lint

var (
        UnMarsharlError = errors.New("Can not unmarshal JSON")
)

type SMDClient struct {
	smdClient *http.Client
	smdBaseURL string
}

func NewSMDClient(baseurl string) *SMDClient {
	c := &http.Client{Timeout: 2 * time.Second}
	return &SMDClient {
		smdClient:  c,
		smdBaseURL: baseurl,
	}
}

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


func (s *SMDClient) IDfromMAC(mac string) (string, error) {
	endpointData := new(sm.ComponentEndpointArray)
	ep := "/hsm/v2/Inventory/ComponentEndpoints/"
	s.getSMD(ep, endpointData)

	for _,ep := range endpointData.ComponentEndpoints {
		id := ep.ID
		nics := ep.RedfishSystemInfo.EthNICInfo
		for _,v := range nics {
			if strings.EqualFold(mac, v.MACAddress) {
				return id, nil
			}
                }
	}
	return "", errors.New("MAC " + mac + " not found for an xname in CompenentEndpoints")
}

func (s *SMDClient) GroupMembership(id string) ([]string, error) {
	ml := new(sm.Membership)
	ep := "/hsm/v2/memberships/" + id
	s.getSMD(ep, ml)
	return ml.GroupLabels, nil
}
