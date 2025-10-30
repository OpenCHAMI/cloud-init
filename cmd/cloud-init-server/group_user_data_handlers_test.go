package main

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenCHAMI/cloud-init/internal/memstore"
	"github.com/OpenCHAMI/cloud-init/internal/smdclient"
	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/go-chi/chi/v5"
)

func TestGroupUserDataHandler_ServesPlainWhenBase64Stored(t *testing.T) {
	store := memstore.NewMemStore()
	f := smdclient.NewFakeSMDClient("test", 2)
	// choose a node and ensure it's in group "compute" per fake client default
	id := f.ListNodes()[0].ID

	content := "#cloud-config\nusers:\n - name: test\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	gd := cistore.GroupData{
		Name: "compute",
		File: cistore.CloudConfigFile{Content: []byte(encoded), Encoding: "base64"},
	}
	if err := store.AddGroupData("compute", gd); err != nil {
		t.Fatalf("failed to add group data: %v", err)
	}

	h := GroupUserDataHandler(f, store)
	r := httptest.NewRequest(http.MethodGet, "/admin/impersonation/"+id+"/compute.yaml", nil)
	w := httptest.NewRecorder()
	// route params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	rctx.URLParams.Add("group", "compute")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	if got := w.Body.String(); got != content {
		t.Fatalf("expected plain content, got: %q", got)
	}
}

func TestGroupUserDataHandler_NotInGroup(t *testing.T) {
	store := memstore.NewMemStore()
	f := smdclient.NewFakeSMDClient("test", 2)
	id := f.ListNodes()[0].ID

	h := GroupUserDataHandler(f, store)
	r := httptest.NewRequest(http.MethodGet, "/admin/impersonation/"+id+"/nonexistent.yaml", nil)
	w := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	rctx.URLParams.Add("group", "nonexistent")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// no helper needed; we attach chi route context inline
