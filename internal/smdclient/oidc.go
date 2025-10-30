package smdclient

import (
	"encoding/json"
	"io"
	"net/http"
)

// Structure of a token response from OIDC server
type oidcTokenData struct {
	AccessToken string `json:"access_token" yaml:"access_token"`
	ExpiresIn   int    `json:"expires_in" yaml:"expires_in"`
	Scope       string `json:"scope" yaml:"scope"`
	TokenType   string `json:"token_type" yaml:"token_type"`
}

// RefreshToken refreshes the cached access token using the configured JWT server.
// TODO: OPAAL returns a token without having to perform the usual OAuth2
// authorization grant. Support for said grant should probably be implemented
// at some point.
func (s *SMDClient) RefreshToken() error {
	// Request new token from OIDC server
	r, err := http.Get(s.tokenEndpoint)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	// Decode server's response to the expected structure
	var tokenResp oidcTokenData
	if err = json.Unmarshal(body, &tokenResp); err != nil {
		return err
	}
	// Extract and store the JWT itself
	s.accessToken = tokenResp.AccessToken
	return nil
}
