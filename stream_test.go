package sse_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

var errDisconnected = errors.New("client disconnected")

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

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	if ct := w.Header().Get("Content-Type"); ct != sse.ContentType {
		t.Errorf("Content-Type: got %q", ct)
	}
}

func TestStream_Send(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

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

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

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

func TestStream_SendHTML(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	err := stream.SendHTML("update", "<ul><li>item</li></ul>")
	if err != nil {
		t.Fatalf("SendHTML: %v", err)
	}

	want := "event: update\ndata: <ul><li>item</li></ul>\n\n"
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

	ctx, cancel := context.WithCancel(context.Background())

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

	ctx, cancel := context.WithCancel(context.Background())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)

	stream := sse.NewStream(w, r)

	done := make(chan struct{})
	go func() {
		defer close(done)
		stream.Heartbeat(ctx, 10*time.Millisecond)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Heartbeat did not stop")
	}

	if !strings.Contains(w.Body.String(), ": heartbeat\n\n") {
		t.Errorf("missing heartbeat in body: %q", w.Body.String())
	}
}

func TestStream_HeartbeatStopsOnCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

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

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

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
		ctx, cancel := context.WithCancel(context.Background())
		r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		stream := sse.NewStream(w, r)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			stream.Heartbeat(ctx, time.Microsecond)
		}()

		go func() {
			defer wg.Done()
			defer cancel()

			for range 10 {
				_ = stream.Send(sse.Event{Event: "ping", Data: "x"})
			}
		}()

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

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

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

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

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

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

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

// TestStream_SendCloseRace verifies that concurrent Send and Close do not race
// or panic. Only the Send+Heartbeat race was previously covered.
func TestStream_SendCloseRace(t *testing.T) {
	t.Parallel()

	for range 20 {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/events", nil)

		stream := sse.NewStream(w, r)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			for range 50 {
				_ = stream.Send(sse.Event{Event: "race", Data: "x"})
			}
		}()

		go func() {
			defer wg.Done()
			_ = stream.Close()
		}()

		wg.Wait()
	}
}

// TestStream_SendHeartbeatCloseRace verifies the three-way race between Send,
// Heartbeat, and Close. All three share the stream mutex; this test ensures no
// panic or data race when they overlap under tight timing.
func TestStream_SendHeartbeatCloseRace(t *testing.T) {
	t.Parallel()

	for range 20 {
		ctx, cancel := context.WithCancel(context.Background())
		r := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		stream := sse.NewStream(w, r)

		var wg sync.WaitGroup
		wg.Add(3)

		go func() {
			defer wg.Done()
			stream.Heartbeat(ctx, time.Microsecond)
		}()

		go func() {
			defer wg.Done()
			defer cancel()

			for range 50 {
				_ = stream.Send(sse.Event{Event: "race", Data: "x"})
			}
		}()

		go func() {
			defer wg.Done()
			_ = stream.Close()
		}()

		wg.Wait()
	}
}
