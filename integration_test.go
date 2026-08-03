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

// TestIntegration_DataStarWireFormat verifies that a DataStar patch-elements
// event composed via SendLines + KeyedLines produces the exact wire bytes that
// DataStar's JS client expects, over a real HTTP round-trip.
func TestIntegration_DataStarWireFormat(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		html := "<div id=\"feed\">\n  <span>1</span>\n</div>"

		_ = stream.SendLines(
			"datastar-patch-elements",
			"selector #feed",
			"mode inner",
			sse.KeyedLines("elements", html),
		)
	}))
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	const want = "event: datastar-patch-elements\n" +
		"data: selector #feed\n" +
		"data: mode inner\n" +
		"data: elements <div id=\"feed\">\n" +
		"data: elements   <span>1</span>\n" +
		"data: elements </div>\n" +
		"\n"

	if string(body) != want {
		t.Errorf("wire format mismatch:\ngot:\n%s\nwant:\n%s", body, want)
	}

	if ct := resp.Header.Get("Content-Type"); ct != sse.ContentType {
		t.Errorf("Content-Type: got %q, want %q", ct, sse.ContentType)
	}
}

// TestIntegration_SubscribeFilter verifies predicate-based filtering over a
// real HTTP round-trip: non-matching broadcasts are never delivered to the
// subscriber's channel; only the matching event reaches the client.
func TestIntegration_SubscribeFilter(t *testing.T) {
	t.Parallel()

	bc := sse.NewBroadcaster[sse.Event]()
	t.Cleanup(bc.Close)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		// Subscribe with filter: only "message" events
		ch := bc.SubscribeFilter(func(evt sse.Event) bool {
			return evt.Event == "message"
		})
		defer bc.Unsubscribe(ch)

		// Send exactly one matching event then exit
		select {
		case <-stream.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok || stream.Send(evt) != nil {
				return
			}
		}
	}))
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	// Deterministic sync: signal on subscribe and unsubscribe
	gotSubscriber := make(chan struct{})
	gotUnsubscribe := make(chan struct{})
	bc.OnSubscribe(func() { close(gotSubscriber) })
	bc.OnUnsubscribe(func() { close(gotUnsubscribe) })

	// Client goroutine: capture the response body
	bodyCh := make(chan string, 1)

	go func() {
		resp, _ := http.Get(server.URL)
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			bodyCh <- string(body)
			_ = resp.Body.Close()
		} else {
			bodyCh <- ""
		}
	}()

	select {
	case <-gotSubscriber:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for subscriber")
	}

	// Broadcast non-matching events — the handler must NOT receive them
	bc.Broadcast(sse.Event{Event: "reaction", Data: "ignored1"})
	bc.Broadcast(sse.Event{Event: "typing", Data: "ignored2"})

	// Broadcast the matching event — the handler receives and sends it
	bc.Broadcast(sse.Event{Event: "message", Data: "filtered-fan-out"})

	// Wait for the handler to send, exit, and run deferred Unsubscribe
	select {
	case <-gotUnsubscribe:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for unsubscribe after filtered broadcast")
	}

	body := <-bodyCh

	// The matching event must be in the wire output
	if !strings.Contains(body, "event: message\n") {
		t.Errorf("missing matching event in body:\n%s", body)
	}

	if !strings.Contains(body, "data: filtered-fan-out\n") {
		t.Errorf("missing matching data in body:\n%s", body)
	}

	// Non-matching events must NOT appear in the wire output
	if strings.Contains(body, "ignored1") {
		t.Errorf("non-matching event 'reaction' leaked through filter:\n%s", body)
	}

	if strings.Contains(body, "ignored2") {
		t.Errorf("non-matching event 'typing' leaked through filter:\n%s", body)
	}

	if count := bc.SubscriberCount(); count != 0 {
		t.Errorf("expected 0 subscribers after handler exit, got %d", count)
	}
}

func TestIntegration_ReplayFiltered(t *testing.T) {
	t.Parallel()

	events := []sse.Event{
		{Event: "message", Data: "msg1", ID: sse.NewEventID("1")},
		{Event: "reaction", Data: "react1", ID: sse.NewEventID("2")},
		{Event: "message", Data: "msg2", ID: sse.NewEventID("3")},
		{Event: "reaction", Data: "react2", ID: sse.NewEventID("4")},
	}

	// Test both store paths: FilteredEventStore (efficient) and plain EventStore (fallback).
	for _, testCase := range []struct {
		name  string
		store sse.EventStore
	}{
		{"filtered_store", &filteredMemoryStore{events: events}},
		{"fallback_store", &memoryStore{events: events}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					stream := sse.NewStream(w, r)
					defer func() { _ = stream.Close() }()

					_, _ = sse.ReplayFiltered(stream, testCase.store, stream.LastEventID(),
						func(evt sse.Event) bool { return evt.Event == "message" })
				}),
			)
			t.Cleanup(func() {
				server.CloseClientConnections()
				server.Close()
			})

			// Client reconnects claiming it last received event "1".
			// Server should replay only "message" events after ID 1 → event 3 (msg2).
			// Event 2 (react1) and 4 (react2) are filtered out.
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			t.Cleanup(cancel)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}

			req.Header.Set("Last-Event-ID", "1")

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

			// Only event 3 (message/msg2) should be replayed after ID 1 with the filter.
			if !strings.Contains(output, "data: msg2") {
				t.Errorf("missing msg2 in %q", output)
			}

			// Reactions must NOT appear — they were filtered out.
			for _, mustNot := range []string{"data: react1", "data: react2"} {
				if strings.Contains(output, mustNot) {
					t.Errorf("filtered-out event %q leaked through: %q", mustNot, output)
				}
			}
		})
	}
}
