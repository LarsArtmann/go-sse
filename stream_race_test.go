package sse_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

func TestStream_SendHeartbeatRaceSafety(t *testing.T) {
	t.Parallel()

	for range 20 {
		ctx, cancel := context.WithCancel(t.Context())
		r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		stream := sse.NewStream(w, r)

		var wg sync.WaitGroup

		wg.Go(func() {
			stream.Heartbeat(ctx, time.Microsecond)
		})

		wg.Go(func() {
			defer cancel()

			for range 10 {
				_ = stream.Send(sse.Event{Event: "ping", Data: "x"})
			}
		})

		wg.Wait()
		_ = stream.Close()
	}
}

func TestStream_SendReturnsErrorOnWriteFailure(t *testing.T) {
	t.Parallel()

	w := &failingResponseWriter{}
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	err := stream.Send(sse.Event{Event: "update", Data: "hello"})
	if err == nil {
		t.Fatal("expected write error from disconnected client, got nil")
	}
}

// TestStream_SendReturnsErrorOnShortWrite pins the partial-write contract: a
// writer that accepts only part of the frame with a nil error must surface
// io.ErrShortWrite (wrapped), never a silent truncation and never a retry —
// re-emitting the frame would put duplicate bytes on the wire.
func TestStream_SendReturnsErrorOnShortWrite(t *testing.T) {
	t.Parallel()

	w := &shortWriteResponseWriter{}
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	err := stream.Send(sse.Event{Event: "update", Data: "hello"})
	if err == nil {
		t.Fatal("expected io.ErrShortWrite from partial-accepting writer, got nil")
	}

	if !errors.Is(err, io.ErrShortWrite) {
		t.Errorf("error wraps io.ErrShortWrite: got %v", err)
	}
}

// TestStream_SendCloseRace verifies that concurrent Send and Close do not race
// or panic. Only the Send+Heartbeat race was previously covered.
func TestStream_SendCloseRace(t *testing.T) {
	t.Parallel()

	for range 20 {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/events", nil)

		stream := sse.NewStream(w, r)

		var wg sync.WaitGroup

		wg.Go(func() {
			for range 50 {
				_ = stream.Send(sse.Event{Event: "race", Data: "x"})
			}
		})

		wg.Go(func() {
			_ = stream.Close()
		})

		wg.Wait()
	}
}

// TestStream_SendHeartbeatCloseRace verifies the three-way race between Send,
// Heartbeat, and Close. All three share the stream mutex; this test ensures no
// panic or data race when they overlap under tight timing.
func TestStream_SendHeartbeatCloseRace(t *testing.T) {
	t.Parallel()

	for range 20 {
		ctx, cancel := context.WithCancel(t.Context())
		r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		stream := sse.NewStream(w, r)

		var wg sync.WaitGroup

		wg.Go(func() {
			stream.Heartbeat(ctx, time.Microsecond)
		})

		wg.Go(func() {
			defer cancel()

			for range 50 {
				_ = stream.Send(sse.Event{Event: "race", Data: "x"})
			}
		})

		wg.Go(func() {
			_ = stream.Close()
		})

		wg.Wait()
	}
}
