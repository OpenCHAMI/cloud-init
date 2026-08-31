// SPDX-FileCopyrightText: 2026 Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build stress

package smdclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	base "github.com/Cray-HPE/hms-base"
	"github.com/stretchr/testify/require"
)

func TestStressComponentInformationCacheHits10K(t *testing.T) {
	const componentCount = 10_000
	var liveLookups atomic.Int64
	client := &SMDClient{
		smdClient:     &http.Client{Transport: failLiveComponentLookupRoundTripper{liveLookups: &liveLookups}},
		smdBaseURL:    "http://smd.example",
		nodesMutex:    &sync.RWMutex{},
		components:    make(map[string]base.Component, componentCount),
		nodes:         make(map[string]NodeMapping),
		ipToXname:     make(map[string]string),
		macToXname:    make(map[string]string),
		wgipToXname:   make(map[string]string),
		accessToken:   "token",
		tokenEndpoint: "http://tokens.example",
	}
	for i := range componentCount {
		id := fmt.Sprintf("x%d", i)
		client.components[id] = base.Component{ID: id, Type: "Node", NID: jsonNumber(i), Role: "compute"}
	}

	var ready sync.WaitGroup
	var start sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(componentCount)
	start.Add(1)
	done.Add(componentCount)

	for i := range componentCount {
		go func() {
			defer done.Done()
			ready.Done()
			start.Wait()

			id := fmt.Sprintf("x%d", i)
			component, err := client.ComponentInformationWithRetry(id, 3)
			if err != nil {
				t.Errorf("ComponentInformationWithRetry(%q) error = %v", id, err)
				return
			}
			if component.ID != id || component.Role != "compute" {
				t.Errorf("component = %+v, want ID %q role compute", component, id)
			}
		}()
	}

	ready.Wait()
	start.Done()
	done.Wait()
	require.Zero(t, liveLookups.Load())
}

func TestStressConcurrentGetSMDCoalescesTokenRefresh10K(t *testing.T) {
	var tokenRequests atomic.Int64
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fresh-token"}`))
	}))
	defer tokenServer.Close()

	client := &SMDClient{
		smdClient:     &http.Client{Transport: tokenAwareRoundTripper{}},
		smdBaseURL:    "http://smd.example",
		tokenEndpoint: tokenServer.URL,
		accessToken:   "stale-token",
		nodesMutex:    &sync.RWMutex{},
	}

	const requestCount = 10_000
	var ready sync.WaitGroup
	var start sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(requestCount)
	start.Add(1)
	done.Add(requestCount)

	for range requestCount {
		go func() {
			defer done.Done()
			ready.Done()
			start.Wait()

			var response map[string]string
			if err := client.getSMD("/component", &response); err != nil {
				t.Errorf("getSMD() error = %v", err)
				return
			}
			if response["ok"] != "true" {
				t.Errorf("response = %v, want ok=true", response)
			}
		}()
	}

	ready.Wait()
	start.Done()
	done.Wait()

	require.Equal(t, int64(1), tokenRequests.Load())
	require.Equal(t, "fresh-token", client.currentAccessToken())
}

type tokenAwareRoundTripper struct{}

func (tokenAwareRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	statusCode := http.StatusOK
	body := `{"ok":"true"}`
	if req.Header.Get("Authorization") != "Bearer fresh-token" {
		statusCode = http.StatusUnauthorized
		body = `{"error":"unauthorized"}`
	}
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

type failLiveComponentLookupRoundTripper struct {
	liveLookups *atomic.Int64
}

func (r failLiveComponentLookupRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r.liveLookups.Add(1)
	return nil, errors.New("unexpected live SMD request during cached component lookup")
}

func jsonNumber(value int) json.Number {
	return json.Number(fmt.Sprintf("%d", value))
}
