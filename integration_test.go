package sse_test

import (
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
