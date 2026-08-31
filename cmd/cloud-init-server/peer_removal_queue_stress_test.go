// SPDX-FileCopyrightText: 2026 Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build stress

package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStressPhoneHomeQueueBackpressure10K(t *testing.T) {
	remover := newBlockingPeerRemover()
	queue := newPeerRemovalQueue(remover, defaultPeerRemovalWorkers, defaultPeerRemovalBuffer)
	handler := PhoneHomeHandler(queue, &phoneHomeSMDClient{})

	for range defaultPeerRemovalWorkers {
		recorder := httptest.NewRecorder()
		handler(recorder, phoneHomeRequest(t))
		if recorder.Code != http.StatusOK {
			t.Fatalf("worker-fill response status = %d, want %d", recorder.Code, http.StatusOK)
		}
	}
	for range defaultPeerRemovalWorkers {
		select {
		case <-remover.started:
		case <-time.After(time.Second):
			t.Fatal("worker did not start removal")
		}
	}

	const requestCount = 10_000
	var okCount atomic.Int64
	var unavailableCount atomic.Int64
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

			recorder := httptest.NewRecorder()
			handler(recorder, phoneHomeRequest(t))
			switch recorder.Code {
			case http.StatusOK:
				okCount.Add(1)
			case http.StatusServiceUnavailable:
				unavailableCount.Add(1)
			default:
				t.Errorf("response status = %d, want %d or %d", recorder.Code, http.StatusOK, http.StatusServiceUnavailable)
			}
		}()
	}

	ready.Wait()
	start.Done()
	done.Wait()

	if got := okCount.Load(); got != defaultPeerRemovalBuffer {
		t.Fatalf("accepted removals = %d, want %d", got, defaultPeerRemovalBuffer)
	}
	if got := unavailableCount.Load(); got != requestCount-defaultPeerRemovalBuffer {
		t.Fatalf("backpressured removals = %d, want %d", got, requestCount-defaultPeerRemovalBuffer)
	}
	close(remover.release)
}
