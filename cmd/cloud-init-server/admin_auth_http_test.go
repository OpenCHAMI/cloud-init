// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
// SPDX-License-Identifier: MIT

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/OpenCHAMI/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

func TestAdminJWTThroughJWKSAndProxy(t *testing.T) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	key, err := jwk.FromRaw(private)
	if err != nil {
		t.Fatal(err)
	}
	if err = key.Set(jwk.KeyIDKey, "fixture-admin-key"); err != nil {
		t.Fatal(err)
	}
	if err = key.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		t.Fatal(err)
	}
	public, err := key.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	set := jwk.NewSet()
	if err = set.AddKey(public); err != nil {
		t.Fatal(err)
	}
	keys := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if encodeErr := json.NewEncoder(w).Encode(set); encodeErr != nil {
			t.Error(encodeErr)
		}
	}))
	t.Cleanup(keys.Close)
	verifier, err := fetchPublicKeyFromURL(keys.URL)
	if err != nil {
		t.Fatal(err)
	}
	token := adminTestToken(t, jwtauth.New("RS256", key, nil), adminTestClaims())
	if err = key.Set(jwk.KeyIDKey, "unknown-fixture-key"); err != nil {
		t.Fatal(err)
	}
	unknownKeyToken := adminTestToken(t, jwtauth.New("RS256", key, nil), adminTestClaims())
	router, data := adminTestRouter(t, verifier)
	backend := httptest.NewServer(router)
	t.Cleanup(backend.Close)
	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	transport := &http.Transport{Proxy: nil}
	t.Cleanup(transport.CloseIdleConnections)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	frontend := httptest.NewServer(proxy)
	t.Cleanup(frontend.Close)
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	for _, entry := range []struct{ name, address string }{{"direct", backend.URL}, {"proxied", frontend.URL}} {
		t.Run(entry.name, func(t *testing.T) {
			for _, test := range []struct {
				name, method, path, token, body string
				status                          int
			}{
				{"anonymous-read", "GET", "/admin/groups", "", "", 401},
				{"invalid-read", "GET", "/admin/groups", "fixture-invalid-token", "", 401},
				{"unknown-key", "GET", "/admin/groups", unknownKeyToken, "", 401},
				{"authenticated-read", "GET", "/admin/groups", token, "", 200},
				{"authenticated-write", "POST", "/admin/cluster-defaults", token, `{"cluster-name":"fixture-authenticated"}`, 201},
				{"anonymous-write", "POST", "/admin/cluster-defaults", "", `{"cluster-name":"unexpected"}`, 401},
				{"impersonation", "GET", "/admin/impersonation/x3000c0b0n1/user-data", token, "", 200},
				{"tokenless-bootstrap", "GET", "/user-data", "", "", 200},
			} {
				t.Run(test.name, func(t *testing.T) {
					request, requestErr := http.NewRequest(test.method, entry.address+test.path, strings.NewReader(test.body))
					if requestErr != nil {
						t.Fatal(requestErr)
					}
					if test.token != "" {
						request.Header.Set("Authorization", "Bearer "+test.token)
					}
					// An attacker-controlled forwarding claim never bypasses authentication.
					request.Header.Set("X-Forwarded-For", "127.0.0.1")
					response, requestErr := client.Do(request)
					if requestErr != nil {
						t.Fatal(requestErr)
					}
					body, requestErr := io.ReadAll(response.Body)
					_ = response.Body.Close()
					if requestErr != nil {
						t.Fatal(requestErr)
					}
					if response.StatusCode != test.status {
						t.Fatalf("got %d, want %d", response.StatusCode, test.status)
					}
					if test.name == "tokenless-bootstrap" && string(body) != "#cloud-config" {
						t.Fatal("bootstrap response changed")
					}
				})
			}
		})
	}
	if data.ClusterDefaults.ClusterName != "fixture-authenticated" {
		t.Fatal("JWT-rejected request changed cluster defaults")
	}
}

func TestAdminJWKSLoadFailureClosesOnlyAdmin(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
	}{
		{"non-success", `{"keys":[]}`, http.StatusServiceUnavailable},
		{"malformed", `not-a-jwks`, http.StatusOK},
		{"empty", `{"keys":[]}`, http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			keys := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			t.Cleanup(keys.Close)
			verifier, err := fetchPublicKeyFromURL(keys.URL)
			if err == nil || verifier != nil {
				t.Fatal("failed JWKS load returned an enabled verifier")
			}
			router, _ := adminTestRouter(t, verifier)
			for _, probe := range []struct {
				path   string
				status int
			}{{"/admin/groups", 503}, {"/user-data", 200}} {
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, probe.path, nil))
				if response.Code != probe.status {
					t.Fatalf("%s: got %d, want %d", probe.path, response.Code, probe.status)
				}
			}
		})
	}
}
