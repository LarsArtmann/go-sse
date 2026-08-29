package ssetest_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
	"github.com/larsartmann/go-sse/ssetest"
)

// sliceStore is a minimal sse.EventStore for the replay dogfood test: it
// returns every stored event whose ID sorts strictly after the request's
// Last-Event-ID, mirroring what a real replay store does on reconnection.
type sliceStore []sse.Event

func (s sliceStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	var after []sse.Event

	for _, evt := range s {
		if evt.ID.Get() > lastID.Get() {
			after = append(after, evt)
		}
	}

	return after, nil
}

// TestE2E_StreamRoundTrip verifies the complete SSE wire format produced by
// go-sse through a real HTTP server and client — event name, multi-line data,
// event ID, and the retry hint all round-trip through the ssetest parser.
func TestE2E_StreamRoundTrip(t *testing.T) {
	t.Parallel()

	handler := streamHandler(func(stream *sse.Stream) {
		_ = stream.Send(sse.Event{
			Event: "feed",
			Data:  "line one\nline two",
			ID:    sse.NewEventID("42"),
			Retry: 3000,
		})
	})

	events := ssetest.Collect(t, handler)
	ssetest.RequireEventCount(t, events, 1)

	ssetest.RequireEventType(t, events[0], "feed")
	ssetest.RequireData(t, events[0], "line one\nline two")
	ssetest.RequireEventID(t, events[0], "42")
	ssetest.RequireRetry(t, events[0], 3000)

	if len(events[0].DataLines) != 2 {
		t.Errorf("datalines: got %d, want 2", len(events[0].DataLines))
	}
}

// broadcastUntilClosed repeatedly broadcasts msg until the hub closes, so a
// consumer test never races the producer past the subscriber connection.
func broadcastUntilClosed(broadcaster *sse.Broadcaster[string], msg string) {
	go func() {
		for {
			broadcaster.Broadcast(msg)
			time.Sleep(2 * time.Millisecond)
		}
	}()
}

// TestE2E_BroadcasterFanOut dogfoods the streaming story: a handler forwarding
// from a real Broadcaster stays open, and CollectN reads exactly N events from
// the live stream.
func TestE2E_BroadcasterFanOut(t *testing.T) {
	t.Parallel()

	broadcaster := sse.NewBroadcaster[string]()
	t.Cleanup(broadcaster.Close)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)

		subscription := broadcaster.Subscribe()
		defer broadcaster.Unsubscribe(subscription)

		for msg := range subscription {
			if err := stream.Send(sse.Event{Event: "feed", Data: msg}); err != nil {
				return
			}
		}
	})

	broadcastUntilClosed(broadcaster, "hello")

	events := ssetest.CollectN(t, handler, 3)
	ssetest.RequireEventCount(t, events, 3)

	for _, evt := range events {
		ssetest.RequireEventType(t, evt, "feed")
		ssetest.RequireData(t, evt, "hello")
	}
}

// TestE2E_ReplayWithLastEventID dogfoods the full reconnection story: the
// handler replays missed events from an EventStore based on the Last-Event-ID
// header, and the test drives the reconnect with WithLastEventID — no browser
// required.
func TestE2E_ReplayWithLastEventID(t *testing.T) {
	t.Parallel()

	store := sliceStore{
		{Event: "feed", Data: "1", ID: sse.NewEventID("1")},
		{Event: "feed", Data: "2", ID: sse.NewEventID("2")},
		{Event: "feed", Data: "3", ID: sse.NewEventID("3")},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		lastID := sse.LastEventIDFromRequest(r)

		if _, err := sse.Replay(stream, store, lastID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		_ = stream.Send(sse.Event{Event: "feed", Data: "live"})
	})

	t.Run("fresh connection replays everything", func(t *testing.T) {
		t.Parallel()

		events := ssetest.Collect(t, handler)
		ssetest.RequireEventCount(t, events, 4) // 3 replayed + 1 live

		ssetest.RequireEventID(t, events[0], "1")
		ssetest.RequireEventID(t, events[2], "3")
		ssetest.RequireData(t, events[3], "live")
	})

	t.Run("reconnect from ID 2 replays only missed events", func(t *testing.T) {
		t.Parallel()

		events := ssetest.Collect(t, handler, ssetest.WithLastEventID("2"))
		ssetest.RequireEventCount(t, events, 2) // event 3 + live

		ssetest.RequireEventID(t, events[0], "3")
		ssetest.RequireData(t, events[1], "live")
	})
}

// TestE2E_HeartbeatIsInvisible verifies that heartbeat comments (": heartbeat")
// keep the connection alive without ever surfacing as events.
func TestE2E_HeartbeatIsInvisible(t *testing.T) {
	t.Parallel()

	handler := streamHandler(func(stream *sse.Stream) {
		_ = stream.Send(sse.Event{Event: "feed", Data: "before"})

		ctx, cancel := context.WithCancel(stream.Context())
		defer cancel()

		done := make(chan struct{})
		go func() {
			defer close(done)
			stream.Heartbeat(ctx, 5*time.Millisecond)
		}()

		time.Sleep(20 * time.Millisecond)
		cancel()
		<-done

		_ = stream.Send(sse.Event{Event: "feed", Data: "after"})
	})

	events := ssetest.Collect(t, handler)
	ssetest.RequireEventCount(t, events, 2)

	ssetest.RequireData(t, events[0], "before")
	ssetest.RequireData(t, events[1], "after")
}

// TestE2E_CollectWithTimeout_Broadcaster proves the timeout path against a
// genuinely open stream: whatever arrived before the deadline is returned.
func TestE2E_CollectWithTimeout_Broadcaster(t *testing.T) {
	t.Parallel()

	broadcaster := sse.NewBroadcaster[string]()
	t.Cleanup(broadcaster.Close)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)

		subscription := broadcaster.Subscribe()
		defer broadcaster.Unsubscribe(subscription)

		for msg := range subscription {
			if err := stream.Send(sse.Event{Event: "feed", Data: msg}); err != nil {
				return
			}
		}
	})

	broadcastUntilClosed(broadcaster, "hello")

	events := ssetest.CollectWithTimeout(t, handler, 150*time.Millisecond)

	if len(events) < 1 {
		t.Fatalf("expected at least 1 event before timeout; got %d", len(events))
	}

	ssetest.RequireData(t, events[0], "hello")
}

// TestE2E_StickyIDSurvivesReconnect pins the full browser-visible ID chain in
// one flow: the server-assigned id: reaches the client (sticky parse state),
// the client echoes it back as Last-Event-ID on reconnect, and — the part
// nothing pinned end-to-end before — events dispatched after a replay still
// report the last seen ID until a new id: arrives, exactly as the browser's
// lastEventId tracking behaves.
func TestE2E_StickyIDSurvivesReconnect(t *testing.T) {
	t.Parallel()

	store := sliceStore{
		{Event: "feed", Data: "1", ID: sse.NewEventID("1")},
		{Event: "feed", Data: "2", ID: sse.NewEventID("2")},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		if _, err := sse.Replay(stream, store, sse.LastEventIDFromRequest(r)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		// Live event with NO id: the wire frame carries none, but the
		// browser-visible ID stays sticky at the last value seen.
		_ = stream.Send(sse.Event{Event: "feed", Data: "live"})
	})

	// First connection: both stored events, then the live one.
	first := ssetest.Collect(t, handler)
	ssetest.RequireEventCount(t, first, 3)
	ssetest.RequireEventID(t, first[0], "1")
	ssetest.RequireEventID(t, first[1], "2")

	// Reconnect from ID 1: event 2 replays, then the live event arrives with
	// no id: on the wire — the parsed event must still report ID "2".
	reconnect := ssetest.Collect(t, handler, ssetest.WithLastEventID("1"))
	ssetest.RequireEventCount(t, reconnect, 2)
	ssetest.RequireEventID(t, reconnect[0], "2")
	ssetest.RequireEventID(t, reconnect[1], "2")
	ssetest.RequireData(t, reconnect[1], "live")
}
