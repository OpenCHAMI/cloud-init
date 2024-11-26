package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

			resp := w.Result()
			assert.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}
