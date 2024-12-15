package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenCHAMI/cloud-init/internal/memstore"
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

type mockStore struct {
	memstore.MemStore
}

func (m *mockStore) Get(id string, groupLabels []string) (citypes.CI, error) {
	if id == "valid-id" {
		return citypes.CI{
			Name: "valid-id",
			CIData: citypes.CIData{
				UserData: map[string]interface{}{"ud-key": "ud-value"},
				MetaData: map[string]interface{}{"md-key": "md-value"},
			},
		}, nil
	}
	return citypes.CI{}, errors.New("not found")
}

type mockSMDClient struct {
	smdclient.SMDClient
}

func (m *mockSMDClient) IDfromMAC(mac string) (string, error) {
	if mac == "valid-mac" {
		return "valid-id", nil
	}
	return "", errors.New("not found")
}

func (m *mockSMDClient) GroupMembership(id string) ([]string, error) {
	if id == "valid-id" {
		return []string{"group1", "group2"}, nil
	}
	return nil, errors.New("not found")
}

func TestGetDataByMAC(t *testing.T) {
	store := &mockStore{}
	sm := &mockSMDClient{}
	handler := NewCiHandler(store, sm)

	tests := []struct {
		name       string
		mac        string
		dataKind   ciDataKind
		statusCode int
	}{
		{
			name:       "Valid MAC",
			mac:        "valid-mac",
			dataKind:   UserData,
			statusCode: http.StatusOK,
		},
		{
			name:       "Invalid MAC",
			mac:        "invalid-mac",
			dataKind:   UserData,
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/harbor/{id}", nil)
			routeCtx := chi.NewRouteContext()
			routeCtx.URLParams.Add("id", tt.mac)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
			w := httptest.NewRecorder()

			handler.GetDataByMAC(tt.dataKind)(w, req)
			t.Logf("Response: %v", w.Body.String())
			resp := w.Result()
			assert.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}

func (m *mockStore) AddGroupData(groupName string, data citypes.GroupData) error {
	if groupName == "existing-group" {
		return errors.New("group already exists")
	}
	return nil
}

func TestAddGroupData(t *testing.T) {
	store := &mockStore{}
	sm := &mockSMDClient{}
	handler := NewCiHandler(store, sm)

	tests := []struct {
		name       string
		groupName  string
		body       string
		statusCode int
	}{
		{
			name:       "Valid Group Data",
			groupName:  "new-group",
			body:       `{"key": "value"}`,
			statusCode: http.StatusOK,
		},
		{
			name:       "Existing Group",
			groupName:  "existing-group",
			body:       `{"key": "value"}`,
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "Invalid JSON",
			groupName:  "new-group",
			body:       `{"key": "value"`,
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "Add rsyslog example",
			groupName:  "rsyslog",
			body:       `{"metadata": [{"key":"value"}], "userdata": {"write_files": [{ "path": "/etc/rsyslog.conf", "content": "rsyslog config" }]}}`,
			statusCode: http.StatusOK,
		},
		{
			name:       "empty user-data",
			groupName:  "rsyslog",
			body:       `{"metadata": {"groups": {"computes": [{"key":"value"}]}}}`,
			statusCode: http.StatusOK,
		},
		{
			name:       "empty meta-data",
			groupName:  "rsyslog",
			body:       `{"user-data": {"write_files": [{ "path": "/etc/rsyslog.conf", "content": "rsyslog config", "group": "rsyslog" }]}}`,
			statusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/harbor/{id}", strings.NewReader(tt.body))
			routeCtx := chi.NewRouteContext()
			routeCtx.URLParams.Add("id", tt.groupName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
			w := httptest.NewRecorder()

			handler.AddGroupData(w, req)
			t.Logf("Response: %v", w.Body.String())
			resp := w.Result()
			assert.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}
