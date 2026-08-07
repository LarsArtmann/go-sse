package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/larsartmann/go-sse"
)

// activityServer holds the broadcaster and event store.
// The broadcaster fans out events to all connected SSE clients; the store
// keeps recent events for reconnection replay.
type activityServer struct {
	broadcaster *sse.Broadcaster[sse.Event]
	store       *memStore
}

func newActivityServer() *activityServer {
	broadcaster := sse.NewBroadcaster[sse.Event]()

	s := &activityServer{
		broadcaster: broadcaster,
		store:       newMemStore(maxStoredEvents),
	}

	// Broadcast the new subscriber count whenever a client connects or
	// disconnects. This gives every tab a live "N clients connected" indicator.
	// We reset $replayed to 0 alongside the count update so the replay banner
	// clears when the subscriber count changes (e.g. on new connection).
	//
	// NOTE: There is a benign TOCTOU race here — SubscriberCount() is read
	// before Broadcast runs, so a concurrent subscribe/unsubscribe could
	// produce a slightly stale count. This is acceptable for a demo; a
	// production system would use an atomic counter or a single-state
	// broadcast event.
	broadcaster.OnSubscribe(func() {
		broadcaster.BroadcastMany(
			replayEvent(0),
			countEvent(broadcaster.SubscriberCount()),
		)
	})
	broadcaster.OnUnsubscribe(func() {
		broadcaster.Broadcast(countEvent(broadcaster.SubscriberCount()))
	})

	return s
}

func (s *activityServer) indexHandler(w http.ResponseWriter, r *http.Request) {
	alertsOnly := r.URL.Query().Get("filter") == "alerts"

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	//nolint:contextcheck // templ uses r.Context() correctly
	if err := indexPage(alertsOnly).Render(r.Context(), w); err != nil {
		log.Printf("render index page: %v", err)
	}
}

func (s *activityServer) eventsHandler(w http.ResponseWriter, r *http.Request) {
	// Allow cross-origin SSE consumption (enables embedding the feed from
	// other domains).
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// NewStream sets the SSE headers (text/event-stream, no-cache, etc.)
	// and writes 200 OK immediately. Do not write to w before this call.
	stream := sse.NewStream(w, r)
	defer func() {
		if err := stream.Close(); err != nil {
			log.Printf("close stream: %v", err)
		}
	}()

	ctx := stream.Context()

	if !s.replayMissedEvents(stream) {
		return
	}

	ch := s.subscribeForRequest(r)
	defer s.broadcaster.Unsubscribe(ch)

	// Heartbeat to keep the connection alive through reverse proxies.
	//
	//nolint:contextcheck // ctx is stream.Context() which is r.Context()
	go stream.Heartbeat(ctx, heartbeatEvery)

	// Event loop: forward broadcast events to the SSE stream.
	// stream.Send serializes on a mutex (http.ResponseWriter is not
	// goroutine-safe), so it's safe to call here while the heartbeat
	// goroutine may also be writing.
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok || stream.Send(evt) != nil {
				return
			}
		}
	}
}

// replayMissedEvents replays events from the store based on the Last-Event-ID
// header and signals the count of replayed events to the client. Returns false
// if a write failed and the handler should abort.
func (s *activityServer) replayMissedEvents(stream *sse.Stream) bool {
	lastID := stream.LastEventID()
	if lastID.IsZero() {
		return true
	}

	// sse.Replay reads EventsAfter from the store and sends each event
	// in order via stream.Send.
	n, err := sse.Replay(stream, s.store, lastID)
	if err != nil {
		log.Printf("replay failed: %v", err)

		return true
	}

	if n == 0 {
		return true
	}

	if err := stream.Send(replayEvent(n)); err != nil {
		return false
	}

	return true
}

// subscribeForRequest registers a subscriber for this request, optionally
// filtered to alerts only. Meta events (subscriber count, replay indicators)
// have no event ID and always pass through the filter so every tab sees the
// subscriber count.
func (s *activityServer) subscribeForRequest(r *http.Request) <-chan sse.Event {
	if r.URL.Query().Get("filter") != "alerts" {
		return s.broadcaster.Subscribe()
	}

	return s.broadcaster.SubscribeFilter(func(evt sse.Event) bool {
		if evt.ID.IsZero() {
			return true
		}

		return strings.Contains(evt.Data, "category "+categoryAlert)
	})
}
