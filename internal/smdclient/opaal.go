package smdclient

import (
	"encoding/json"
	"io"
	"net/http"
)

// Structure of a token reponse from OPAAL
type opaalTokenData struct {
	Access_token string `json:"access_token"`
	Expires_in   int    `json:"expires_in"`
	Scope        string `json:"scope"`
	Token_type   string `json:"token_type"`
}

// Refresh the cached access token, using the provided JWT server
func (s *SMDClient) RefreshToken() error {
	// Request new token from OPAAL
	r, err := http.Get(s.tokenServer + "/token")
	if err != nil {
		return err
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	// Decode OPAAL's response to the expected structure
	var tokenResp opaalTokenData
	if err = json.Unmarshal(body, &tokenResp); err != nil {
		return err
	}
	// Extract and store the JWT itself
	s.accessToken = tokenResp.Access_token
	return nil
}
