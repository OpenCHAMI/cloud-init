// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smdclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Structure of a token reponse from OIDC server
type oidcTokenData struct {
	AccessToken string `json:"access_token" yaml:"access_token"`
	ExpiresIn   int    `json:"expires_in" yaml:"expires_in"`
	Scope       string `json:"scope" yaml:"scope"`
	TokenType   string `json:"token_type" yaml:"token_type"`
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
	if s.tokenClient == nil {
		return fmt.Errorf("token HTTP client is not configured")
	}

	// Request new token from OIDC server using the provided context.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.tokenEndpoint, nil)
	if err != nil {
		return fmt.Errorf("creating token request: %w", err)
	}
	r, err := s.tokenClient.Do(req)
	if err != nil {
		return fmt.Errorf("requesting token: %w", err)
	}
	defer r.Body.Close()
	if r.StatusCode < http.StatusOK || r.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("token endpoint returned HTTP %d", r.StatusCode)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("reading token response: %w", err)
	}
	// Decode server's response to the expected structure
	var tokenResp oidcTokenData
	if err = json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("decoding token response: %w", err)
	}
	token := strings.TrimSpace(tokenResp.AccessToken)
	if token == "" {
		return fmt.Errorf("token response contains an empty access token")
	}

	// Store the JWT safely.
	s.accessTokenMutex.Lock()
	s.accessToken = token
	s.accessTokenMutex.Unlock()
	return nil
}
