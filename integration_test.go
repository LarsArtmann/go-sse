package sse_test

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

func TestIntegration_DirectSendAndHeaders(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		_ = stream.Send(sse.Event{Event: "update", Data: "hello-world", Retry: 3000})
		_ = stream.Close()
	}))
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()

	if ct := resp.Header.Get("Content-Type"); ct != sse.ContentType {
		t.Errorf("Content-Type: got %q, want %q", ct, sse.ContentType)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	output := string(body)
	if !strings.Contains(output, "event: update\n") {
		t.Errorf("missing event line in %q", output)
	}

	if !strings.Contains(output, "data: hello-world\n") {
		t.Errorf("missing data line in %q", output)
	}

	if !strings.Contains(output, "retry: 3000\n") {
		t.Errorf("missing retry line in %q", output)
	}

	if !strings.HasSuffix(output, "\n\n") {
		t.Errorf("output must end with \\n\\n: %q", output)
	}
}

func TestIntegration_BroadcasterFanOut(t *testing.T) {
	t.Parallel()

	bc := sse.NewBroadcaster[sse.Event]()
	t.Cleanup(bc.Close)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		ch := bc.Subscribe()
		defer bc.Unsubscribe(ch)

		// Send exactly one event then exit
		select {
		case <-stream.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}

			_ = stream.Send(evt)
		}
	}))
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	// Deterministic sync: signal on the first subscribe and the first unsubscribe.
	gotSubscriber := make(chan struct{})
	gotUnsubscribe := make(chan struct{})
	bc.OnSubscribe(func() { close(gotSubscriber) })
	bc.OnUnsubscribe(func() { close(gotUnsubscribe) })

	go func() {
		resp, _ := http.Get(server.URL)
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}()

	select {
	case <-gotSubscriber:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for subscriber")
	}

	bc.Broadcast(sse.Event{Event: "ping", Data: "fan-out-test"})

	// Wait deterministically for the handler to send the event, exit, and run its
	// deferred Unsubscribe — no time.Sleep, so no flakiness under scheduler pressure.
	select {
	case <-gotUnsubscribe:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for unsubscribe after broadcast")
	}

	if count := bc.SubscriberCount(); count != 0 {
		t.Errorf("expected 0 subscribers after handler exit, got %d", count)
	}
}

// TestIntegration_HeartbeatDelivery verifies that SSE comment-frame heartbeats
// are delivered over a real HTTP round-trip — the proxy-survival path that keeps
// idle connections alive through Nginx/Cloudflare/AWS ALB idle timeouts.
func TestIntegration_HeartbeatDelivery(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		// Emit keep-alive comment frames on a tight cadence so the client observes
		// several within the test's deadline.
		go stream.Heartbeat(r.Context(), 20*time.Millisecond)

		<-r.Context().Done()
	}))
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	t.Cleanup(cancel)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	t.Cleanup(func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	})

	// Read the stream line-by-line and count heartbeat comment frames received
	// over the wire. Three frames proves the keep-alive path works end-to-end;
	// the context deadline turns a silent failure into a clear test error.
	reader := bufio.NewReader(resp.Body)

	const want = 3

	var heartbeats int

	for heartbeats < want {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf(
				"read heartbeat %d: %v (expected %d comment frames over the wire)",
				heartbeats+1,
				err,
				want,
			)
		}

		if line == ": heartbeat\n" {
			heartbeats++
		}
	}
}

// TestIntegration_LastEventIDReconnectionReplay verifies the core SSE
// reconnection use case over a real HTTP round-trip: the client reconnects with
// a Last-Event-ID header and the server replays exactly the events it missed.
func TestIntegration_LastEventIDReconnectionReplay(t *testing.T) {
	t.Parallel()

	store := &memoryStore{
		events: []sse.Event{
			{Event: "item", Data: "first", ID: sse.NewEventID("1")},
			{Event: "item", Data: "second", ID: sse.NewEventID("2")},
			{Event: "item", Data: "third", ID: sse.NewEventID("3")},
			{Event: "item", Data: "fourth", ID: sse.NewEventID("4")},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		// Reconnect scenario: replay every event strictly after the Last-Event-ID.
		if _, err := sse.Replay(stream, store, stream.LastEventID()); err != nil {
			return
		}
	}))
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	// Client reconnects claiming it last received event "2" → server replays 3 and 4.
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	t.Cleanup(cancel)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	req.Header.Set("Last-Event-ID", "2")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	_ = resp.Body.Close()

	output := string(body)

	// Events 3 and 4 must be replayed; 1 and 2 must NOT appear (already received).
	for _, want := range []string{"id: 3\n", "data: third\n", "id: 4\n", "data: fourth\n"} {
		if !strings.Contains(output, want) {
			t.Errorf("missing replayed fragment %q in %q", want, output)
		}
	}

	for _, mustNot := range []string{"data: first", "data: second"} {
		if strings.Contains(output, mustNot) {
			t.Errorf("already-received event %q must not be replayed: %q", mustNot, output)
		}
	}
}
