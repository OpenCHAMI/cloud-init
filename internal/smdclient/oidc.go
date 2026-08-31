// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smdclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
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
	// Serialize refresh to avoid concurrent token fetches.
	s.refreshLock.Lock()
	defer s.refreshLock.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), defaultRefreshTimeout)
	defer cancel()
	return s.refreshTokenWithContext(ctx)
}

const defaultRefreshTimeout = 10 * time.Second

func (s *SMDClient) refreshTokenIfCurrent(rejectedToken string) error {
	// Fast path: if token already different, nothing to do.
	s.accessTokenMutex.Lock()
	if s.accessToken != rejectedToken {
		s.accessTokenMutex.Unlock()
		return nil
	}
	s.accessTokenMutex.Unlock()

	// Serialize refresh to avoid concurrent token fetches.
	s.refreshLock.Lock()
	defer s.refreshLock.Unlock()

	// Re-check token after acquiring lock (it may have been refreshed by another goroutine).
	s.accessTokenMutex.Lock()
	if s.accessToken != rejectedToken {
		s.accessTokenMutex.Unlock()
		return nil
	}
	s.accessTokenMutex.Unlock()

	// Acquire new token with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), defaultRefreshTimeout)
	defer cancel()
	return s.refreshTokenWithContext(ctx)
}

func (s *SMDClient) refreshTokenWithContext(ctx context.Context) error {
	// Request new token from OIDC server using the provided context.
	req, err := http.NewRequestWithContext(ctx, "GET", s.tokenEndpoint, nil)
	if err != nil {
		return err
	}
	r, err := http.DefaultClient.Do(req)
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
	// Store the JWT safely.
	s.accessTokenMutex.Lock()
	s.accessToken = tokenResp.Access_token
	s.accessTokenMutex.Unlock()
	return nil
}
