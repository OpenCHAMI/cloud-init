// SPDX-FileCopyrightText: 2026 Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smdclient

import (
	"encoding/json"
	"io"
	"net/http"
)

// Structure of a token reponse from OIDC server
type oidcTokenData struct {
	Access_token string `json:"access_token" yaml:"access_token"`
	Expires_in   int    `json:"expires_in" yaml:"expires_in"`
	Scope        string `json:"scope" yaml:"scope"`
	Token_type   string `json:"token_type" yaml:"token_type"`
}

// Refresh the cached access token, using the provided JWT server
// TODO: OPAAL returns a token without having to perform the usual OAuth2
// authorization grant. Support for said grant should probably be implemented
// at some point.
func (s *SMDClient) RefreshToken() error {
	s.accessTokenMutex.Lock()
	defer s.accessTokenMutex.Unlock()
	return s.refreshTokenLocked()
}

func (s *SMDClient) refreshTokenIfCurrent(rejectedToken string) error {
	s.accessTokenMutex.Lock()
	defer s.accessTokenMutex.Unlock()
	if s.accessToken != rejectedToken {
		return nil
	}
	return s.refreshTokenLocked()
}

func (s *SMDClient) refreshTokenLocked() error {
	// Request new token from OIDC server
	r, err := http.Get(s.tokenEndpoint)
	if err != nil {
		return err
	}
	defer r.Body.Close()
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
	s.accessToken = tokenResp.Access_token
	return nil
}
