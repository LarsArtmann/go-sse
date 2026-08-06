// Package main implements a DataStar-compatible SSE server using go-sse.
//
// Demonstrates the full go-sse feature set through a live activity feed:
//   - Broadcaster: multi-client fan-out (open multiple tabs to see it)
//   - SubscriberCount: live count via OnSubscribe/OnUnsubscribe callbacks
//   - SubscribeFilter: "alerts only" view via ?filter=alerts query param
//   - EventStore + Replay: missed events replayed on reconnect
//   - Heartbeat: keeps connections alive through proxies
//   - KeyedLines/SendLines: DataStar keyed-data-line wire format
//
// Run: go run ./example/datastar/
// Then: open http://localhost:8765 in your browser
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"math/rand/v2"
	"net/http"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/larsartmann/go-sse"
)

const (
	datastarAddr    = ":8765"
	emitInterval    = 2 * time.Second
	heartbeatEvery  = 15 * time.Second
	maxStoredEvents = 50
	shutdownTimeout = 5 * time.Second
)

//go:embed all:static
var staticFiles embed.FS

// --- EventStore ---

// memStore is an in-memory ring buffer implementing sse.EventStore.
// It keeps the last maxStoredEvents events for reconnection replay.
type memStore struct {
	mu     sync.RWMutex
	events []sse.Event
	cap    int
}

func newMemStore(capacity int) *memStore {
	return &memStore{
		events: make([]sse.Event, 0, capacity),
		cap:    capacity,
	}
}

func (s *memStore) Append(evt sse.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, evt)

	if len(s.events) > s.cap {
		s.events = s.events[len(s.events)-s.cap:]
	}
}

func (s *memStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	lastSeq, err := strconv.Atoi(lastID.Get())
	if err != nil {
		lastSeq = 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []sse.Event

	for _, evt := range s.events {
		seq, err := strconv.Atoi(evt.ID.Get())
		if err != nil {
			continue
		}

		if seq > lastSeq {
			result = append(result, evt)
		}
	}

	return result, nil
}

// --- Activity Server ---

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
	broadcaster.OnSubscribe(func() {
		broadcaster.Broadcast(countEvent(broadcaster.SubscriberCount()))
	})
	broadcaster.OnUnsubscribe(func() {
		broadcaster.Broadcast(countEvent(broadcaster.SubscriberCount()))
	})

	return s
}

// startProducer runs a background goroutine that emits a random activity
// event every emitInterval. Each event gets a monotonically increasing ID
// so reconnecting clients can replay missed events.
func (s *activityServer) startProducer(ctx context.Context) {
	var id int64

	emit := func() {
		id++

		item := generateItem()

		evt := feedItemEvent(id, item)

		s.store.Append(evt)
		s.broadcaster.Broadcast(evt)
	}

	emit() // emit one immediately so the user sees something fast

	ticker := time.NewTicker(emitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			emit()
		}
	}
}

// --- Event Generators ---

// activityItem is a single entry in the activity feed.
type activityItem struct {
	category string // "info", "alert", "success"
	badge    string // "INFO", "ALERT", "OK"
	message  string
	time     string
}

// msgTemplate pairs a display category with a random message generator.
type msgTemplate struct {
	category string
	badge    string
	gen      func() string
}

var msgTemplates = []msgTemplate{
	{"alert", "ALERT", func() string { return fmt.Sprintf("CPU usage above 90%% on node-%d", rand.IntN(20)+1) }},
	{"alert", "ALERT", func() string { return fmt.Sprintf("Disk space below 10%% on node-%d", rand.IntN(20)+1) }},
	{"alert", "ALERT", func() string { return fmt.Sprintf("Memory leak detected in service-%d", rand.IntN(50)+1) }},
	{"alert", "ALERT", func() string { return fmt.Sprintf("Response time exceeding SLA on endpoint-%d", rand.IntN(10)+1) }},
	{"success", "OK", func() string { return fmt.Sprintf("Deploy #%d passed all checks", rand.IntN(999)+1) }},
	{"success", "OK", func() string { return fmt.Sprintf("Migration v1.%d.%d applied successfully", rand.IntN(9), rand.IntN(9)) }},
	{"success", "OK", func() string { return fmt.Sprintf("Build #%d completed in %ds", rand.IntN(999)+1, rand.IntN(30)+1) }},
	{"success", "OK", func() string { return fmt.Sprintf("Health check passed for service-%d", rand.IntN(50)+1) }},
	{"info", "INFO", func() string { return fmt.Sprintf("User session-%d started", rand.IntN(9999)+1) }},
	{"info", "INFO", func() string { return fmt.Sprintf("Cache invalidated for region-%d", rand.IntN(10)+1) }},
	{"info", "INFO", func() string { return fmt.Sprintf("Scheduled task-%d completed", rand.IntN(500)+1) }},
	{"info", "INFO", func() string { return fmt.Sprintf("Configuration reloaded for service-%d", rand.IntN(50)+1) }},
}

func generateItem() activityItem {
	t := msgTemplates[rand.IntN(len(msgTemplates))]

	return activityItem{
		category: t.category,
		badge:    t.badge,
		message:  t.gen(),
		time:     time.Now().Format("15:04:05"),
	}
}

// feedItemEvent builds a DataStar patch-elements SSE event that prepends
// a single feed item to the #feed div. The event carries a sequential ID
// so it can be replayed on reconnection.
func feedItemEvent(id int64, item activityItem) sse.Event {
	data := strings.Join([]string{
		"selector #feed",
		"mode prepend",
		sse.KeyedLines("elements", feedItemHTML(item)),
	}, "\n")

	return sse.Event{
		Event: "datastar-patch-elements",
		Data:  data,
		ID:    sse.NewEventID(strconv.FormatInt(id, 10)),
	}
}

// countEvent builds a DataStar patch-signals event that updates the
// subscriberCount signal. No event ID — this is an ephemeral meta event
// that should not be replayed.
func countEvent(count int) sse.Event {
	return sse.Event{
		Event: "datastar-patch-signals",
		Data:  sse.KeyedLines("signals", fmt.Sprintf(`{"subscriberCount":%d}`, count)),
	}
}

// replayEvent builds a DataStar patch-signals event that sets the replayed
// signal, triggering the "Replayed N missed events" banner.
func replayEvent(n int) sse.Event {
	return sse.Event{
		Event: "datastar-patch-signals",
		Data:  sse.KeyedLines("signals", fmt.Sprintf(`{"replayed":%d}`, n)),
	}
}

func feedItemHTML(item activityItem) string {
	return fmt.Sprintf(
		`<div class="feed-item feed-item--%s"><span class="feed-item__badge">%s</span><span class="feed-item__message">%s</span><span class="feed-item__time">%s</span></div>`,
		item.category,
		item.badge,
		item.message,
		item.time,
	)
}

// --- HTTP Handlers ---

func (s *activityServer) indexHandler(w http.ResponseWriter, r *http.Request) {
	alertsOnly := r.URL.Query().Get("filter") == "alerts"

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := indexPage(alertsOnly).Render(r.Context(), w); err != nil {
		log.Printf("render index page: %v", err)
	}
}

func (s *activityServer) eventsHandler(w http.ResponseWriter, r *http.Request) {
	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	ctx := stream.Context()

	// Replay missed events on reconnect (Last-Event-ID header).
	if lastID := stream.LastEventID(); !lastID.IsZero() {
		if n, err := sse.Replay(stream, s.store, lastID); err != nil {
			log.Printf("replay failed: %v", err)
		} else if n > 0 {
			if err := stream.Send(replayEvent(n)); err != nil {
				return
			}
		}
	}

	// Subscribe — optionally filtered to alerts only.
	// Meta events (subscriber count, replay indicators) have no event ID and
	// always pass through the filter so every tab sees the subscriber count.
	var ch <-chan sse.Event

	if r.URL.Query().Get("filter") == "alerts" {
		ch = s.broadcaster.SubscribeFilter(func(evt sse.Event) bool {
			if evt.ID.IsZero() {
				return true
			}

			return strings.Contains(evt.Data, "feed-item--alert")
		})
	} else {
		ch = s.broadcaster.Subscribe()
	}

	defer s.broadcaster.Unsubscribe(ch)

	// Heartbeat to keep the connection alive through reverse proxies.
	go stream.Heartbeat(ctx, heartbeatEvery)

	// Event loop: forward broadcast events to the SSE stream.
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

// --- Main ---

func main() {
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("static sub FS: %v", err)
	}

	server := newActivityServer()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go server.startProducer(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", server.indexHandler)
	mux.HandleFunc("GET /events", server.eventsHandler)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	httpServer := &http.Server{
		Addr:    datastarAddr,
		Handler: mux,
	}

	go func() {
		log.Printf("DataStar example on http://localhost%s", datastarAddr)
		log.Print("Open the URL in multiple browser tabs to see real-time fan-out.")
		log.Fatal(httpServer.ListenAndServe()) //nolint:gosec // G114: intentional for example server
	}()

	<-ctx.Done()
	log.Print("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	_ = httpServer.Shutdown(shutdownCtx)
	_ = server.broadcaster.Shutdown(shutdownCtx)
}
