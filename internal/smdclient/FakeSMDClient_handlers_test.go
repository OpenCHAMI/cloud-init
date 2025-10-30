package smdclient

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	base "github.com/Cray-HPE/hms-base"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/go-chi/chi/v5"
)

func TestAddNodeToInventoryHandler(t *testing.T) {
	f := NewFakeSMDClient("test", 2)
	// Build request body
	node := OpenCHAMINodeWithGroups{
		OpenCHAMIComponent: cistore.OpenCHAMIComponent{
			Component: base.Component{ID: "x9c0b0n0"},
			MAC:       "AA:BB:CC:DD:EE:FF",
			IP:        "10.0.0.9",
		},
		Groups: []string{"extra", "compute"},
	}
	b, _ := json.Marshal(node)
	r := httptest.NewRequest(http.MethodPost, "/admin/nodes", bytes.NewReader(b))
	w := httptest.NewRecorder()

	h := AddNodeToInventoryHandler(f)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc == "" {
		t.Errorf("expected Location header to be set")
	}

	// Verify the node shows up in the list endpoint
	lr := httptest.NewRequest(http.MethodGet, "/admin/nodes", nil)
	lw := httptest.NewRecorder()
	ListNodesHandler(f).ServeHTTP(lw, lr)
	if lw.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from list, got %d", lw.Code)
	}
}

func TestUpdateNodeHandler(t *testing.T) {
	f := NewFakeSMDClient("test", 2)
	// pick an existing node id
	id := f.rosettaMapping[0].ComponentID

	update := OpenCHAMINodeWithGroups{
		OpenCHAMIComponent: cistore.OpenCHAMIComponent{
			MAC: "AA:BB:CC:00:00:01",
			IP:  "10.0.0.10",
		},
		Groups: []string{"special"},
	}
	b, _ := json.Marshal(update)
	r := httptest.NewRequest(http.MethodPut, "/admin/nodes/"+id, bytes.NewReader(b))
	// inject chi route param into context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h := UpdateNodeHandler(f)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	// Confirm group added
	if !containsString(f.groups["special"], id) {
		t.Errorf("expected node %s to be added to group 'special'", id)
	}
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
