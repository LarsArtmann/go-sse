package sse_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestStream_Send(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	err := stream.Send(sse.Event{Event: "update", Data: "<div>new</div>"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	want := "event: update\ndata: <div>new</div>\n\n"
	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
	}
}

func TestStream_SendMultiple(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	_ = stream.Send(sse.Event{Event: "e1", Data: "d1"})
	_ = stream.Send(sse.Event{Event: "e2", Data: "d2"})

	body := w.Body.String()
	if !strings.Contains(body, "event: e1\ndata: d1\n\n") {
		t.Errorf("missing e1: %q", body)
	}

	if !strings.Contains(body, "event: e2\ndata: d2\n\n") {
		t.Errorf("missing e2: %q", body)
	}
}

func TestStream_SendData(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	err := stream.SendData("update", "<ul><li>item</li></ul>")
	if err != nil {
		t.Fatalf("SendData: %v", err)
	}

	want := "event: update\ndata: <ul><li>item</li></ul>\n\n"
	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
	}
}

func TestStream_SendLines(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	html := "<div>\n  <span>hi</span>\n</div>"

	err := stream.SendLines(
		"datastar-patch-elements",
		"selector #feed",
		"mode inner",
		sse.KeyedLines("elements", html),
	)
	if err != nil {
		t.Fatalf("SendLines: %v", err)
	}

	want := "event: datastar-patch-elements\n" +
		"data: selector #feed\n" +
		"data: mode inner\n" +
		"data: elements <div>\n" +
		"data: elements   <span>hi</span>\n" +
		"data: elements </div>\n" +
		"\n"

	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
	}
}

func TestStream_SendKeyed(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	err := stream.SendKeyed("datastar-patch-signals", "signals", `{"progress":50}`)
	if err != nil {
		t.Fatalf("SendKeyed: %v", err)
	}

	want := "event: datastar-patch-signals\n" +
		"data: signals {\"progress\":50}\n" +
		"\n"

	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
	}
}

func TestStream_SendKeyed_MultiLine(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	err := stream.SendKeyed("datastar-patch-elements", "elements", "<div>\n</div>")
	if err != nil {
		t.Fatalf("SendKeyed: %v", err)
	}

	want := "event: datastar-patch-elements\n" +
		"data: elements <div>\n" +
		"data: elements </div>\n" +
		"\n"

	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
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

func TestStream_DoubleCloseSafety(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)

	_ = stream.Close()
	_ = stream.Close() // must not panic
}

func TestStream_SendJSON(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	type payload struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}

	err := stream.SendJSON("update", payload{Name: "go-sse", N: 7})
	if err != nil {
		t.Fatalf("SendJSON: %v", err)
	}

	want := `event: update` + "\n" + `data: {"name":"go-sse","n":7}` + "\n\n"
	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
	}
}

func TestStream_SendJSON_MarshalError(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	// Channels cannot be JSON-marshalled.
	err := stream.SendJSON("bad", make(chan int))
	if err == nil {
		t.Fatal("expected marshal error, got nil")
	}

	if w.Body.Len() != 0 {
		t.Errorf("nothing should be written on marshal failure, got %q", w.Body.String())
	}
}

func TestStream_SendJSON_NilValue(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	err := stream.SendJSON("clear", nil)
	if err != nil {
		t.Fatalf("SendJSON(nil): %v", err)
	}

	want := "event: clear\ndata: null\n\n"
	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
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
