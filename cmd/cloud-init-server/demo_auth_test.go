// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var adminDemoScripts = []string{
	"set_cluster_defaults.sh", "add_compute_group.sh", "add_cabinet_groups.sh",
	"override_hostname.sh", "add_node.sh", "update_x3003c3b5n0.sh",
}

func demoEnvironment() []string {
	var environment []string
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "CLOUD_INIT_ADMIN_TOKEN=") && !strings.HasPrefix(entry, "CLOUD_INIT_URL=") {
			environment = append(environment, entry)
		}
	}
	return environment
}

func TestDemoScriptsRequireAdminTokenBeforeRequests(t *testing.T) {
	for _, script := range adminDemoScripts {
		t.Run(script, func(t *testing.T) {
			directory := t.TempDir()
			marker := filepath.Join(directory, "curl-called")
			// Never contact a service while exercising missing credentials.
			stub := "#!/bin/sh\nprintf invoked >> \"$DEMO_TEST_CURL_LOG\"\n"
			if err := os.WriteFile(filepath.Join(directory, "curl"), []byte(stub), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("bash", filepath.Join("..", "..", "demo", script))
			command.Env = append(demoEnvironment(), "PATH="+directory+string(os.PathListSeparator)+os.Getenv("PATH"), "DEMO_TEST_CURL_LOG="+marker)
			if err := command.Run(); err == nil {
				t.Error("demo accepted a missing administrative token")
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Error("demo attempted a request without an administrative token")
			}
		})
	}
}

// Run the bundled clients against a disposable loopback server, never a deployment.
func TestDemoScriptsAuthenticatedRequests(t *testing.T) {
	for _, script := range adminDemoScripts {
		for _, status := range []int{http.StatusCreated, http.StatusUnauthorized, http.StatusServiceUnavailable} {
			t.Run(script+"/"+http.StatusText(status), func(t *testing.T) {
				const token = "fixture-admin-token"
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.Header.Get("Authorization") != "Bearer "+token {
						t.Error("missing bearer header")
					}
					if !strings.HasPrefix(r.URL.Path, "/fixture/admin/") {
						t.Errorf("incorrect API base: %s", r.URL.Path)
					}
					if r.Method != http.MethodPost && r.Method != http.MethodPut {
						t.Errorf("unexpected method: %s", r.Method)
					}
					var body map[string]interface{}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("invalid JSON body: %v", err)
					}
					w.WriteHeader(status)
				}))
				defer server.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, "bash", filepath.Join("..", "..", "demo", script))
				command.Env = append(demoEnvironment(), "CLOUD_INIT_ADMIN_TOKEN="+token, "CLOUD_INIT_URL="+server.URL+"/fixture/", "NO_PROXY=*", "no_proxy=*")
				output, err := command.CombinedOutput()
				if status == http.StatusCreated && err != nil {
					t.Fatalf("demo failed: %v: %s", err, output)
				}
				if status != http.StatusCreated && err == nil {
					t.Error("demo accepted HTTP failure")
				}
				if strings.Contains(string(output), token) {
					t.Error("demo printed the token")
				}
				want := 1
				if status == http.StatusCreated && (script == "add_cabinet_groups.sh" || script == "update_x3003c3b5n0.sh") {
					want = 4
				}
				if int(requests.Load()) != want {
					t.Errorf("got %d requests, want %d (stop on first failure)", requests.Load(), want)
				}
			})
		}
	}
}
