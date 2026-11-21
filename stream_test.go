package sse_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

var errDisconnected = errors.New("client disconnected")

// recordingResponseWriter is a concurrency-safe http.ResponseWriter that records
// all writes and signals each one via wrote. httptest.ResponseRecorder is not
// safe for concurrent use, so this type lets a test observe Heartbeat writes
// from another goroutine without a data race.
type recordingResponseWriter struct {
	header http.Header
	mu     sync.Mutex
	body   bytes.Buffer
	wrote  chan struct{}
}

func newRecordingResponseWriter() *recordingResponseWriter {
	return &recordingResponseWriter{wrote: make(chan struct{}, 256)}
}

func (r *recordingResponseWriter) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}

	return r.header
}

func (r *recordingResponseWriter) Write(p []byte) (int, error) {
	r.mu.Lock()
	n, err := r.body.Write(p)
	r.mu.Unlock()

	if err == nil {
		select {
		case r.wrote <- struct{}{}:
		default:
		}
	}

	return n, err
}

func (r *recordingResponseWriter) WriteHeader(int) {}

func (r *recordingResponseWriter) Body() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.body.String()
}

func TestSetHeaders(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()

	sse.SetHeaders(w)

	if ct := w.Header().Get("Content-Type"); ct != sse.ContentType {
		t.Errorf("Content-Type: got %q, want %q", ct, sse.ContentType)
	}

	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control: got %q", cc)
	}

	if conn := w.Header().Get("Connection"); conn != "keep-alive" {
		t.Errorf("Connection: got %q", conn)
	}
}

func TestNewStream_SetsHeadersAndStatus(t *testing.T) {
	t.Parallel()

	_, w := newTestStream(t)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	if ct := w.Header().Get("Content-Type"); ct != sse.ContentType {
		t.Errorf("Content-Type: got %q", ct)
	}
}

func TestStream_Context(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(t.Context())
	w := httptest.NewRecorder()

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	if stream.Context() != r.Context() {
		t.Error("Context mismatch")
	}
}

func TestStream_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	cancel()

	select {
	case <-stream.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("context not cancelled")
	}
}

func TestStream_OnDisconnect(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)

	called := false
	stream.OnDisconnect(func() { called = true })

	_ = stream.Close()

	if !called {
		t.Fatal("OnDisconnect callback not called")
	}
}

func TestStream_OnDisconnectMultipleInOrder(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)

	var order []int
	stream.OnDisconnect(func() { order = append(order, 1) })
	stream.OnDisconnect(func() { order = append(order, 2) })
	stream.OnDisconnect(func() { order = append(order, 3) })

	_ = stream.Close()

	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("order: %v", order)
	}
}

func TestStream_LastEventID(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	r.Header.Set("Last-Event-ID", "evt-99")

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	if id := stream.LastEventID(); id.Get() != "evt-99" {
		t.Errorf("LastEventID: got %q", id.Get())
	}
}

func TestStream_LastEventID_Empty(t *testing.T) {
	t.Parallel()

	stream, _ := newTestStream(t)

	if id := stream.LastEventID(); !id.IsZero() {
		t.Errorf("LastEventID: expected zero, got %q", id.Get())
	}
}

func TestLastEventIDFromRequest(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	r.Header.Set("Last-Event-ID", "evt-42")

	id := sse.LastEventIDFromRequest(r)
	if id.Get() != "evt-42" {
		t.Errorf("got %q", id.Get())
	}
}

func TestLastEventIDFromRequest_MaliciousInputTreatedAsEmpty(t *testing.T) {
	t.Parallel()

	cases := []string{"evt\nmalicious", "evt\rmalicious", "evt\r\nmalicious", "\n", "\r"}

	for _, header := range cases {
		r := httptest.NewRequest(http.MethodGet, "/events", nil)
		r.Header.Set("Last-Event-ID", header)

		id := sse.LastEventIDFromRequest(r)
		if !id.IsZero() {
			t.Errorf("Last-Event-ID %q should be rejected as zero, got %q", header, id.Get())
		}
	}
}

// failingResponseWriter is an http.ResponseWriter whose Write always errors,
// simulating a client that has disconnected / a broken pipe.
type failingResponseWriter struct {
	header http.Header
}

func (f *failingResponseWriter) Header() http.Header {
	if f.header == nil {
		f.header = make(http.Header)
	}

	return f.header
}

func (f *failingResponseWriter) Write(_ []byte) (int, error) { return 0, errDisconnected }
func (f *failingResponseWriter) WriteHeader(int)             {}

var _ http.ResponseWriter = (*failingResponseWriter)(nil)

// shortWriteResponseWriter is an http.ResponseWriter that accepts only part
// of each Write (n < len(p), nil error) — a writer violating the io.Writer
// contract's full-delivery expectation without reporting an error. It stands
// in for wrapped/buggy ResponseWriters (compression, buffering middleware).
type shortWriteResponseWriter struct {
	header http.Header
}

func (s *shortWriteResponseWriter) Header() http.Header {
	if s.header == nil {
		s.header = make(http.Header)
	}

	return s.header
}

func (s *shortWriteResponseWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	return len(p) / 2, nil
}

func (s *shortWriteResponseWriter) WriteHeader(int) {}

var _ http.ResponseWriter = (*shortWriteResponseWriter)(nil)

// TestStream_RequestContextCancelMidStream drives r.Context() cancellation
// against a real socket with Sends in flight: the client cancels mid-stream,
// and the handler must tear down cleanly — stream.Context() observes the
// cancellation, the send loop exits without deadlocking, and no panic escapes
// the handler (net/http would swallow a handler panic in its own recover, so
// panics are forwarded to the test explicitly).
func TestStream_RequestContextCancelMidStream(t *testing.T) {
	t.Parallel()

	handlerDone := make(chan error, 1)
	handlerPanic := make(chan any, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				handlerPanic <- p
				close(handlerDone)
			}
		}()

		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		var err error
	loop:
		for {
			select {
			case <-stream.Context().Done():
				break loop
			default:
			}

			if err = stream.Send(sse.Event{Event: "tick", Data: "x"}); err != nil {
				break loop
			}
		}

		handlerDone <- stream.Context().Err()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/events", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("client Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Cancel while the handler is mid-send-loop. The timer starts only
	// after Do returned (connection established, handler streaming), so
	// the cancellation always lands mid-stream — a deadline measured
	// from request creation instead raced connection setup against the
	// 50ms budget and flaked under parallel-suite load.
	time.AfterFunc(50*time.Millisecond, cancel)
	select {
	case p := <-handlerPanic:
		t.Fatalf("handler panicked on context cancellation: %v", p)
	case ctxErr := <-handlerDone:
		if ctxErr == nil {
			t.Error("stream context was not cancelled when the client went away")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("handler did not return after context cancellation (deadlock?)")
	}
}
