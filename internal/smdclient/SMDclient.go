package smdclient

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"log"

	"github.com/OpenCHAMI/smd/v2/pkg/sm"
	"github.com/golang-jwt/jwt"
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
	smdClient   *http.Client
	smdBaseURL  string
	accessToken string
}

// NewSMDClient creates a new SMDClient which connects to the SMD server at baseurl
// and uses the provided JWT for authentication
func NewSMDClient(baseurl string, jwt string) *SMDClient {
	c := &http.Client{Timeout: 2 * time.Second}
	return &SMDClient{
		smdClient:   c,
		smdBaseURL:  baseurl,
		accessToken: jwt,
	}
}

// getSMD is a helper function to initialize the SMDClient
func (s *SMDClient) getSMD(ep string, smd interface{}) error {
	url := s.smdBaseURL + ep
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if s.accessToken != "" {
		//validate the JWT without verifying the signature
		//if the JWT is not valid, the request will fail
		token, _, err := new(jwt.Parser).ParseUnverified(s.accessToken, jwt.MapClaims{})
		if err != nil {
			return errors.New("poorly formed JWT: " + err.Error())
		}
		log.Println("Loaded JWT token:", s.accessToken)
		log.Println("Claims:", token.Claims)
		req.Header.Set("Authorization", "Bearer "+s.accessToken)
	} else {
		return errors.New("poorly formed JWT")
	}
	resp, err := s.smdClient.Do(req)
	if err != nil {
		return err
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
	return "", errors.New("MAC " + mac + " not found for an xname in ComponentEndpoints")
}

// GroupMembership returns the group labels for the xname with the given ID
func (s *SMDClient) GroupMembership(id string) ([]string, error) {
	ml := new(sm.Membership)
	ep := "/hsm/v2/memberships/" + id
	s.getSMD(ep, ml)
	return ml.GroupLabels, nil
}
