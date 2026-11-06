// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
// SPDX-License-Identifier: MIT

package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/OpenCHAMI/jwtauth/v5"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/openchami/cloud-init/internal/memstore"
	"github.com/openchami/cloud-init/internal/smdclient"
	"github.com/openchami/cloud-init/pkg/cistore"
)

const adminTestSigningKey = "fixture-admin-signing-key"

type adminRouteCase struct {
	method string
	path   string
}

var adminRouteCases = []adminRouteCase{
	{http.MethodGet, "/admin/cluster-defaults"},
	{http.MethodPost, "/admin/cluster-defaults"},
	{http.MethodPut, "/admin/instance-info/x3000c0b0n1"},
	{http.MethodGet, "/admin/groups"},
	{http.MethodPost, "/admin/groups"},
	{http.MethodGet, "/admin/groups/compute"},
	{http.MethodPut, "/admin/groups/compute"},
	{http.MethodDelete, "/admin/groups/compute"},
	{http.MethodGet, "/admin/impersonation/x3000c0b0n1/user-data"},
	{http.MethodGet, "/admin/impersonation/x3000c0b0n1/meta-data"},
	{http.MethodGet, "/admin/impersonation/x3000c0b0n1/vendor-data"},
	{http.MethodGet, "/admin/impersonation/x3000c0b0n1/compute.yaml"},
	{http.MethodGet, "/admin/fake-sm/nodes"},
	{http.MethodPost, "/admin/fake-sm/nodes"},
	{http.MethodPut, "/admin/fake-sm/nodes/x3000c0b0n1"},
}

func adminTestRouter(t *testing.T, keyset *jwtauth.JWTAuth) (*chi.Mux, *memstore.MemStore) {
	t.Helper()
	oldImpersonation, oldFake, oldWG := impersonationEnabled, fakeSMDEnabled, wireGuardMiddleware
	t.Cleanup(func() {
		impersonationEnabled, fakeSMDEnabled, wireGuardMiddleware = oldImpersonation, oldFake, oldWG
	})
	impersonationEnabled, fakeSMDEnabled, wireGuardMiddleware = true, true, nil
	data := memstore.NewMemStore()
	if err := data.AddGroupData("compute", cistore.GroupData{
		Name: "compute", File: cistore.CloudConfigFile{Content: []byte("#cloud-config\nfixture: true\n")},
	}); err != nil {
		t.Fatal(err)
	}
	handler := &CiHandler{store: data, sm: smdclient.NewFakeSMDClient("fixture", 1)}
	router := chi.NewRouter()
	router.Use(middleware.StripSlashes)
	initCiClientRouter(router, handler, nil, nil)
	initCiAdminRouter(router, handler, keyset)
	return router, data
}

func adminTestClaims() map[string]interface{} {
	return map[string]interface{}{
		"sub": "fixture-operator", "iss": "https://issuer.invalid", "aud": "cloud-init-admin",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
}

func adminTestToken(t *testing.T, signer *jwtauth.JWTAuth, claims map[string]interface{}) string {
	t.Helper()
	_, token, err := signer.Encode(claims)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestAdminRoutesDenyWhenJWTVerifierUnavailable(t *testing.T) {
	router, _ := adminTestRouter(t, nil)
	for _, route := range adminRouteCases {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(route.method, route.path, nil))
			if response.Code != http.StatusServiceUnavailable {
				t.Fatalf("admin route without a JWT verifier: got %d, want 503", response.Code)
			}
		})
	}
}

func TestAdminJWTExistingClaimProfile(t *testing.T) {
	// Configured signing keys grant admin authority; audience values and exp
	// presence are not enforced by the existing claim profile.
	keyset := jwtauth.New("HS256", []byte(adminTestSigningKey), nil)
	router, _ := adminTestRouter(t, keyset)
	claims := adminTestClaims()
	claims["aud"] = "another-service"
	delete(claims, "exp")
	request := httptest.NewRequest(http.MethodGet, "/admin/groups", nil)
	request.Header.Set("Authorization", "Bearer "+adminTestToken(t, keyset, claims))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("inherited JWT claim profile changed: got %d, want 200", response.Code)
	}
}

func TestAdminRoutesRequireJWT(t *testing.T) {
	keyset := jwtauth.New("HS256", []byte(adminTestSigningKey), nil)
	expired := adminTestClaims()
	expired["exp"] = time.Now().Add(-time.Hour).Unix()
	notYetValid := adminTestClaims()
	notYetValid["nbf"] = time.Now().Add(time.Hour).Unix()
	cases := []struct {
		name  string
		token string
	}{
		{name: "missing"},
		{name: "malformed", token: "fixture-not-a-jwt"},
		{name: "wrong-signature", token: adminTestToken(t, jwtauth.New("HS256", []byte("different-key"), nil), adminTestClaims())},
		{name: "wrong-algorithm", token: adminTestToken(t, jwtauth.New("HS384", []byte(adminTestSigningKey), nil), adminTestClaims())},
		{name: "expired", token: adminTestToken(t, keyset, expired)},
		{name: "not-yet-valid", token: adminTestToken(t, keyset, notYetValid)},
	}
	for _, claim := range []string{"sub", "iss", "aud"} {
		claims := adminTestClaims()
		delete(claims, claim)
		cases = append(cases, struct{ name, token string }{"missing-" + claim, adminTestToken(t, keyset, claims)})
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			router, data := adminTestRouter(t, keyset)
			for _, route := range adminRouteCases {
				t.Run(route.method+" "+route.path, func(t *testing.T) {
					request := httptest.NewRequest(route.method, route.path, strings.NewReader(`{"name":"unexpected","cluster-name":"changed"}`))
					if test.token != "" {
						request.Header.Set("Authorization", "Bearer "+test.token)
					}
					request.Header.Set("X-Forwarded-For", "127.0.0.1")
					response := httptest.NewRecorder()
					router.ServeHTTP(response, request)
					if response.Code != http.StatusUnauthorized {
						t.Fatalf("got %d, want 401", response.Code)
					}
				})
			}
			if len(data.GetGroups()) != 1 || data.ClusterDefaults.ClusterName != "" || len(data.Instances) != 0 {
				t.Fatal("rejected admin requests modified state")
			}
		})
	}
}

func TestAdminRoutesRejectAmbientCredentials(t *testing.T) {
	keyset := jwtauth.New("HS256", []byte(adminTestSigningKey), nil)
	token := adminTestToken(t, keyset, adminTestClaims())
	router, _ := adminTestRouter(t, keyset)
	for _, name := range []string{"cookie", "query"} {
		t.Run(name, func(t *testing.T) {
			path := "/admin/groups"
			if name == "query" {
				path += "?jwt=" + token
			}
			request := httptest.NewRequest(http.MethodGet, path, nil)
			if name == "cookie" {
				request.AddCookie(&http.Cookie{Name: "jwt", Value: token})
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("%s token: got %d, want 401", name, response.Code)
			}
		})
	}
}

func TestAdminRouteInventory(t *testing.T) {
	router, _ := adminTestRouter(t, nil)
	var actual []string
	if err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.HasPrefix(route, "/admin/") {
			actual = append(actual, method+" "+route)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	expected := []string{
		"GET /admin/cluster-defaults", "POST /admin/cluster-defaults",
		"PUT /admin/instance-info/{id}", "GET /admin/groups", "POST /admin/groups",
		"GET /admin/groups/{id}", "PUT /admin/groups/{name}", "DELETE /admin/groups/{id}",
		"GET /admin/impersonation/{id}/user-data", "GET /admin/impersonation/{id}/meta-data",
		"GET /admin/impersonation/{id}/vendor-data", "GET /admin/impersonation/{id}/{group}.yaml",
		"GET /admin/fake-sm/nodes", "POST /admin/fake-sm/nodes", "PUT /admin/fake-sm/nodes/{id}",
	}
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("admin route inventory changed:\ngot %v\nwant %v", actual, expected)
	}
	if len(adminRouteCases) != len(expected) {
		t.Fatal("admin rejection coverage does not match registered route inventory")
	}
}

func TestAdminJWTAllowsExistingOperations(t *testing.T) {
	keyset := jwtauth.New("HS256", []byte(adminTestSigningKey), nil)
	token := adminTestToken(t, keyset, adminTestClaims())
	router, data := adminTestRouter(t, keyset)
	steps := []struct {
		name, method, path, body string
		status                   int
	}{
		{"cluster-write", "POST", "/admin/cluster-defaults", `{"cluster-name":"fixture"}`, 201},
		{"cluster-read", "GET", "/admin/cluster-defaults", "", 200},
		{"instance-write", "PUT", "/admin/instance-info/x3000c0b0n1", `{"instance-id":"fixture-instance"}`, 201},
		{"group-create", "POST", "/admin/groups", `{"name":"fixture-created"}`, 201},
		{"group-update", "PUT", "/admin/groups/fixture-created", `{"name":"fixture-created","description":"updated"}`, 201},
		{"group-read", "GET", "/admin/groups/fixture-created", "", 200},
		{"groups-read", "GET", "/admin/groups", "", 200},
		{"impersonation", "GET", "/admin/impersonation/x3000c0b0n1/user-data", "", 200},
		{"impersonation-group", "GET", "/admin/impersonation/x3000c0b0n1/compute.yaml", "", 200},
		{"simulator-read", "GET", "/admin/fake-sm/nodes", "", 200},
		{"group-delete", "DELETE", "/admin/groups/fixture-created", "", 200},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			request := httptest.NewRequest(step.method, step.path, strings.NewReader(step.body))
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != step.status {
				t.Fatalf("got %d, want %d: %s", response.Code, step.status, response.Body.String())
			}
		})
	}
	if data.ClusterDefaults.ClusterName != "fixture" || len(data.Instances) != 1 || len(data.GetGroups()) != 1 {
		t.Fatal("authenticated admin operations did not retain expected stored data")
	}
}

func TestAdminAuthPreservesTokenlessBootstrap(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run(map[bool]string{false: "no-jwks", true: "configured-jwks"}[configured], func(t *testing.T) {
			var keyset *jwtauth.JWTAuth
			if configured {
				keyset = jwtauth.New("HS256", []byte(adminTestSigningKey), nil)
			}
			router, _ := adminTestRouter(t, keyset)
			for _, route := range []struct {
				name, method, path, body string
				status                   int
			}{
				{"user-data", "GET", "/user-data", "", 200},
				{"group-data", "GET", "/compute.yaml", "", 200},
				{"meta-data", "GET", "/meta-data", "", 200},
				{"vendor-data", "GET", "/vendor-data", "", 200},
				{"phone-home", "POST", "/phone-home/x3000c0b0n1", "", 200},
				// Invalid enrollment remains a normal400, not an operator JWT challenge.
				{"wg-init", "POST", "/wg-init", `{}`, 400},
			} {
				t.Run(route.name, func(t *testing.T) {
					request := httptest.NewRequest(route.method, route.path, strings.NewReader(route.body))
					request.RemoteAddr = "10.20.30.1:1234"
					response := httptest.NewRecorder()
					router.ServeHTTP(response, request)
					if response.Code != route.status {
						t.Fatalf("got %d, want %d: %s", response.Code, route.status, response.Body.String())
					}
					if route.path == "/meta-data" && response.Body.Len() == 0 {
						t.Fatal("bootstrap metadata payload disappeared")
					}
				})
			}
		})
	}
}
