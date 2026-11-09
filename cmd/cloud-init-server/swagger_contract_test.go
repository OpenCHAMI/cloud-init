// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestSwaggerAdminAuthenticationContract(t *testing.T) {
	data, err := os.ReadFile("../../docs/swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Security            []map[string][]string `json:"security"`
		SecurityDefinitions map[string]struct {
			Type        string `json:"type"`
			In          string `json:"in"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"securityDefinitions"`
		Paths map[string]map[string]struct {
			Security  []map[string][]string      `json:"security"`
			Responses map[string]json.RawMessage `json:"responses"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	scheme := spec.SecurityDefinitions["BearerAuth"]
	if scheme.Type != "apiKey" || scheme.In != "header" || scheme.Name != "Authorization" || !strings.Contains(scheme.Description, "Bearer") {
		t.Errorf("missing Authorization-header Bearer JWT security scheme: %+v", scheme)
	}
	if len(spec.Security) != 0 {
		t.Error("global security would challenge public bootstrap routes")
	}
	router, _ := adminTestRouter(t, nil)
	adminCount := 0
	if err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.HasPrefix(route, "/admin/") {
			return nil
		}
		adminCount++
		operation, ok := spec.Paths[route][strings.ToLower(method)]
		if !ok {
			t.Errorf("undocumented admin route: %s %s", method, route)
			return nil
		}
		if !reflect.DeepEqual(operation.Security, []map[string][]string{{"BearerAuth": {}}}) {
			t.Errorf("%s %s must require BearerAuth, got %v", method, route, operation.Security)
		}
		for _, status := range []string{"401", "503"} {
			if _, ok := operation.Responses[status]; !ok {
				t.Errorf("%s %s missing %s authentication response", method, route, status)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if adminCount != 15 {
		t.Fatalf("admin inventory changed: got %d operations, want 15", adminCount)
	}
	for _, route := range []struct{ method, path string }{
		{"get", "/user-data"}, {"get", "/meta-data"}, {"get", "/vendor-data"},
		{"get", "/{group}.yaml"}, {"post", "/phone-home/{id}"}, {"post", "/wg-init"},
	} {
		if _, ok := spec.Paths[route.path][route.method]; !ok {
			t.Errorf("missing public bootstrap documentation: %s %s", route.method, route.path)
		}
	}
	for path, methods := range spec.Paths {
		if strings.HasPrefix(path, "/admin/") {
			continue
		}
		for method, operation := range methods {
			if len(operation.Security) != 0 {
				t.Errorf("public %s %s must not require operator authentication", method, path)
			}
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("public /openapi.json: got %d", response.Code)
	}
	var checked, served map[string]interface{}
	if err := json.Unmarshal(data, &checked); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(response.Body.Bytes(), &served); err != nil {
		t.Fatal(err)
	}
	// The runtime template emits empty dynamic fields omitted in swagger.json.
	for _, key := range []string{"host", "basePath", "schemes"} {
		if _, ok := checked[key]; !ok {
			if value := served[key]; value == nil || reflect.DeepEqual(value, "") || reflect.DeepEqual(value, []interface{}{}) {
				delete(served, key)
			}
		}
	}
	if !reflect.DeepEqual(checked, served) {
		t.Error("served /openapi.json differs from checked-in swagger.json")
	}
}

func TestSwaggerSimulatorNodeSchemas(t *testing.T) {
	data, err := os.ReadFile("../../docs/swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	type schema struct {
		Ref        string                     `json:"$ref"`
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Items      json.RawMessage            `json:"items"`
	}
	var spec struct {
		Definitions map[string]schema `json:"definitions"`
		Paths       map[string]map[string]struct {
			Parameters []struct {
				In     string `json:"in"`
				Schema schema `json:"schema"`
			} `json:"parameters"`
			Responses map[string]struct {
				Schema schema `json:"schema"`
			} `json:"responses"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []struct{ method, path string }{
		{"post", "/admin/fake-sm/nodes"}, {"put", "/admin/fake-sm/nodes/{id}"},
		{"get", "/admin/fake-sm/nodes"},
	} {
		t.Run(operation.method+" "+operation.path, func(t *testing.T) {
			entry := spec.Paths[operation.path][operation.method]
			var ref string
			if operation.method == "get" {
				response := entry.Responses["200"].Schema
				if response.Type != "array" {
					t.Fatalf("node list must be an array, got %q", response.Type)
				}
				var item schema
				if err := json.Unmarshal(response.Items, &item); err != nil {
					t.Fatal(err)
				}
				ref = item.Ref
			} else {
				for _, parameter := range entry.Parameters {
					if parameter.In == "body" {
						ref = parameter.Schema.Ref
					}
				}
			}
			if !strings.HasPrefix(ref, "#/definitions/") {
				t.Fatalf("missing simulator node schema reference: %q", ref)
			}
			node := spec.Definitions[strings.TrimPrefix(ref, "#/definitions/")]
			for _, field := range []string{"ID", "mac", "ip", "wgip"} {
				var property schema
				if err := json.Unmarshal(node.Properties[field], &property); err != nil {
					t.Errorf("missing or invalid node field %s in %s", field, ref)
				} else if property.Type != "string" {
					t.Errorf("node field %s must be a string, got %q", field, property.Type)
				}
			}
			var nid schema
			if err := json.Unmarshal(node.Properties["NID"], &nid); err != nil {
				t.Error("missing or invalid NID field")
			} else if nid.Type != "number" {
				t.Errorf("NID must match json.Number numeric serialization, got %q", nid.Type)
			}
			if operation.method != "get" {
				var groups schema
				if err := json.Unmarshal(node.Properties["groups"], &groups); err != nil {
					t.Fatal("missing or invalid groups request field")
				}
				var item schema
				if err := json.Unmarshal(groups.Items, &item); err != nil {
					t.Fatal(err)
				}
				if groups.Type != "array" || item.Type != "string" {
					t.Error("groups must be an array of strings")
				}
			}
		})
	}
}
