// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package main

// Adapted from OpenCHAMI SMD's auth.go

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	jwtauth "github.com/OpenCHAMI/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	openchami_authenticator "github.com/openchami/chi-middleware/auth"
)

// adminJWTMiddleware protects every route registered under /admin. An unavailable
// verifier closes only that API, so tokenless node bootstrap can keep running.
func adminJWTMiddleware(keyset *jwtauth.JWTAuth) func(http.Handler) http.Handler {
	if keyset == nil {
		return func(_ http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "admin JWT verification is not configured or unavailable", http.StatusServiceUnavailable)
			})
		}
	}
	return func(next http.Handler) http.Handler {
		// Administrative APIs accept bearer headers, not ambient browser cookies.
		return jwtauth.Verify(keyset, jwtauth.TokenFromHeader)(
			openchami_authenticator.AuthenticatorWithRequiredClaims(keyset, []string{"sub", "iss", "aud"})(next),
		)
	}
}

type statusCheckTransport struct {
	http.RoundTripper
}

func (ct *statusCheckTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err == nil && resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return resp, err
}

func newHTTPClient() *http.Client {
	return &http.Client{Transport: &statusCheckTransport{}, Timeout: 10 * time.Second}
}

func fetchPublicKeyFromURL(url string) (*jwtauth.JWTAuth, error) {
	client := newHTTPClient()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	set, err := jwk.Fetch(ctx, url, jwk.WithHTTPClient(client))
	if err != nil {
		msg := "%w"

		// if the error tree contains an EOF, it means that the response was empty,
		// so add a more descriptive message to the error tree
		if errors.Is(err, io.EOF) {
			msg = "received empty response for key: %w"
		}

		return nil, fmt.Errorf(msg, err)
	}
	if set.Len() == 0 {
		return nil, errors.New("JWKS contains no verification keys")
	}
	jwks, err := json.Marshal(set)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JWKS: %v", err)
	}
	keyset, err := jwtauth.NewKeySet(jwks)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize JWKS: %v", err)
	}

	return keyset, nil
}
