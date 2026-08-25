package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/openchami/cloud-init/internal/smdclient"
)

type blockingPeerRemover struct {
	started chan struct{}
	release chan struct{}
	removed chan string
}

func newBlockingPeerRemover() *blockingPeerRemover {
	return &blockingPeerRemover{
		started: make(chan struct{}, 16),
		release: make(chan struct{}),
		removed: make(chan string, 16),
	}
}

func (r *blockingPeerRemover) RemovePeer(peerName string) error {
	r.started <- struct{}{}
	<-r.release
	r.removed <- peerName
	return nil
}

type phoneHomeSMDClient struct {
	smdclient.FakeSMDClient
}

func (phoneHomeSMDClient) IDfromIP(string) (string, error) {
	return "x0c0s0b0n0", nil
}

func (phoneHomeSMDClient) IPfromID(string) (string, error) {
	return "10.1.0.1", nil
}

func TestPeerRemovalQueueBoundsWork(t *testing.T) {
	remover := newBlockingPeerRemover()
	queue := newPeerRemovalQueue(remover, 1, 1)

	if !queue.TryEnqueue("peer-1") {
		t.Fatal("first enqueue unexpectedly failed")
	}
	select {
	case <-remover.started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start first removal")
	}
	if !queue.TryEnqueue("peer-2") {
		t.Fatal("buffered enqueue unexpectedly failed")
	}
	if queue.TryEnqueue("peer-3") {
		t.Fatal("enqueue succeeded when worker and buffer were saturated")
	}

	close(remover.release)
	for range 2 {
		select {
		case <-remover.removed:
		case <-time.After(time.Second):
			t.Fatal("queued removal did not finish")
		}
	}
}

func TestPhoneHomeHandlerReturnsUnavailableWhenRemovalQueueFull(t *testing.T) {
	remover := newBlockingPeerRemover()
	queue := newPeerRemovalQueue(remover, 1, 1)
	handler := PhoneHomeHandler(queue, &phoneHomeSMDClient{})

	first := httptest.NewRecorder()
	handler(first, phoneHomeRequest(t))
	if first.Code != http.StatusOK {
		t.Fatalf("first response status = %d, want %d", first.Code, http.StatusOK)
	}
	select {
	case <-remover.started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start first removal")
	}

	second := httptest.NewRecorder()
	handler(second, phoneHomeRequest(t))
	if second.Code != http.StatusOK {
		t.Fatalf("second response status = %d, want %d", second.Code, http.StatusOK)
	}

	third := httptest.NewRecorder()
	handler(third, phoneHomeRequest(t))
	if third.Code != http.StatusServiceUnavailable {
		t.Fatalf("third response status = %d, want %d", third.Code, http.StatusServiceUnavailable)
	}
	close(remover.release)
}

func TestPhoneHomeHandlerWithoutWireGuardStillReturnsOK(t *testing.T) {
	handler := PhoneHomeHandler(nil, &phoneHomeSMDClient{})
	recorder := httptest.NewRecorder()
	handler(recorder, phoneHomeRequest(t))
	if recorder.Code != http.StatusOK {
		t.Fatalf("response status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func phoneHomeRequest(t *testing.T) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/phone-home/x0c0s0b0n0", nil)
	r.RemoteAddr = "10.1.0.1:12345"
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "x0c0s0b0n0")
	return r.WithContext(contextWithRoute(r.Context(), rctx))
}

func contextWithRoute(ctx context.Context, rctx *chi.Context) context.Context {
	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}
