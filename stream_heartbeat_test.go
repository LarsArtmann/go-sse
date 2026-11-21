package sse_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

func TestStream_Heartbeat(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	// recordingResponseWriter is concurrency-safe; httptest.ResponseRecorder is
	// not, and the Heartbeat goroutine writes while the test reads.
	w := newRecordingResponseWriter()
	r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

	stream := sse.NewStream(w, r)

	done := make(chan struct{})
	go func() {
		defer close(done)
		stream.Heartbeat(ctx, 10*time.Millisecond)
	}()

	// Wait deterministically for the first heartbeat write, then cancel — no
	// time.Sleep, so the test never flakes under scheduler pressure.
	select {
	case <-w.wrote:
	case <-time.After(2 * time.Second):
		t.Fatal("Heartbeat did not write a frame within 2s")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Heartbeat did not stop")
	}

	if !strings.Contains(w.Body(), ": heartbeat\n\n") {
		t.Errorf("missing heartbeat in body: %q", w.Body())
	}
}

func TestStream_HeartbeatStopsOnCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

	stream := sse.NewStream(w, r)

	done := make(chan struct{})
	go func() {
		defer close(done)
		stream.Heartbeat(ctx, 10*time.Millisecond)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Heartbeat did not stop on cancel")
	}
}

// TestStream_HeartbeatStopsAfterClose pins the closed-stream exit path: once
// the handler's deferred Close has run, a Heartbeat goroutine must stop on its
// own — even with a live context. Without this, a tick racing handler teardown
// writes into the finished ResponseWriter and panics inside net/http (the
// 2026-09-03 red-master CI failure in TestIntegration_HeartbeatDelivery).
func TestStream_HeartbeatStopsAfterClose(t *testing.T) {
	t.Parallel()

	w := newRecordingResponseWriter()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)

	done := make(chan struct{})
	go func() {
		defer close(done)
		stream.Heartbeat(t.Context(), time.Millisecond)
	}()

	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Heartbeat did not stop after Close")
	}

	bodyAfterClose := w.Body()

	time.Sleep(5 * time.Millisecond)

	if body := w.Body(); body != bodyAfterClose {
		t.Errorf("Heartbeat wrote after Close: got %q, want %q", body, bodyAfterClose)
	}
}

// TestStream_HeartbeatExitsOnWriteError covers the write-failure exit path:
// when the underlying writer errors, Heartbeat must return instead of looping.
func TestStream_HeartbeatExitsOnWriteError(t *testing.T) {
	t.Parallel()

	w := &failingResponseWriter{}
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)

	done := make(chan struct{})
	go func() {
		defer close(done)
		stream.Heartbeat(t.Context(), time.Millisecond)
	}()

	// failingResponseWriter.Write always errors, so the first tick's
	// WriteHeartbeat fails and Heartbeat returns — proving the error-exit path.
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Heartbeat did not exit after write error")
	}
}
