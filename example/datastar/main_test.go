package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

// --- memStore unit tests ---

func TestMemStore_EmptyStore(t *testing.T) {
	t.Parallel()

	store := newMemStore(maxStoredEvents)

	events, err := store.EventsAfter(sse.NewEventID("0"))
	if err != nil {
		t.Fatalf("EventsAfter on empty store: unexpected error: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("EventsAfter on empty store: expected 0 events, got %d", len(events))
	}
}

func TestMemStore_InvalidID(t *testing.T) {
	t.Parallel()

	store := newMemStore(maxStoredEvents)
	store.Append(makeTestEvent(1))
	store.Append(makeTestEvent(2))

	tests := []struct {
		name  string
		idStr string
		want  int
	}{
		{"non-integer", "abc", 2},
		{"empty string", "", 2},
		{"very large ID", "999999", 0},
		{"zero ID gets all", "0", 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			events, err := store.EventsAfter(sse.NewEventID(tc.idStr))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(events) != tc.want {
				t.Errorf("id=%q: expected %d events, got %d", tc.idStr, tc.want, len(events))
			}
		})
	}
}

func TestMemStore_SequentialReplay(t *testing.T) {
	t.Parallel()

	store := newMemStore(maxStoredEvents)

	for i := range 5 {
		store.Append(makeTestEvent(int64(i + 1)))
	}

	events, err := store.EventsAfter(sse.NewEventID("2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events after ID 2, got %d", len(events))
	}

	for i, evt := range events {
		expectedID := strconv.FormatInt(int64(i+3), 10)
		if evt.ID.Get() != expectedID {
			t.Errorf("event %d: expected ID %s, got %s", i, expectedID, evt.ID.Get())
		}
	}
}

func TestMemStore_RingBufferEviction(t *testing.T) {
	t.Parallel()

	capacity := 5
	store := newMemStore(capacity)

	total := capacity + 10
	for i := range total {
		store.Append(makeTestEvent(int64(i + 1)))
	}

	events, err := store.EventsAfter(sse.NewEventID("0"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != capacity {
		t.Fatalf("expected %d events after eviction, got %d", capacity, len(events))
	}

	firstID := events[0].ID.Get()
	expectedFirst := "11"
	if firstID != expectedFirst {
		t.Errorf("expected first event ID %s after eviction, got %s", expectedFirst, firstID)
	}

	lastID := events[len(events)-1].ID.Get()
	expectedLast := "15"
	if lastID != expectedLast {
		t.Errorf("expected last event ID %s after eviction, got %s", expectedLast, lastID)
	}
}

// --- Concurrent fan-out test ---

func TestConcurrentFanOut(t *testing.T) {
	t.Parallel()

	server := newActivityServer()
	defer server.broadcaster.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.startProducer(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", server.indexHandler)
	mux.HandleFunc("GET /events", server.eventsHandler)

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	const collectDuration = 6 * time.Second

	var (
		eventsA, eventsB []string
		errA, errB       error
		wg               sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		eventsA, errA = collectSSEEvents(httpServer.URL+"/events", collectDuration)
	}()

	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond) // slight offset so both overlap
		eventsB, errB = collectSSEEvents(httpServer.URL+"/events", collectDuration)
	}()

	wg.Wait()

	if errA != nil {
		t.Fatalf("client A error: %v", errA)
	}

	if errB != nil {
		t.Fatalf("client B error: %v", errB)
	}

	if len(eventsA) == 0 {
		t.Fatal("client A received 0 events — fan-out or producer is broken")
	}

	if len(eventsB) == 0 {
		t.Fatal("client B received 0 events — fan-out or producer is broken")
	}

	idsA := extractEventIDs(eventsA)
	idsB := extractEventIDs(eventsB)

	if len(idsA) == 0 {
		t.Fatal("client A received 0 events with IDs — expected feed-item events")
	}

	common := commonIDs(idsA, idsB)
	if len(common) == 0 {
		t.Fatalf(
			"no common event IDs between clients A and B — fan-out is broken\nA: %v\nB: %v",
			idsA,
			idsB,
		)
	}

	t.Logf("fan-out verified: %d events to client A, %d to client B, %d shared IDs",
		len(idsA), len(idsB), len(common))
}

func TestSubscriberCountIncrements(t *testing.T) {
	t.Parallel()

	server := newActivityServer()
	defer server.broadcaster.Close()

	if count := server.broadcaster.SubscriberCount(); count != 0 {
		t.Fatalf("expected 0 subscribers at start, got %d", count)
	}

	server.broadcaster.OnSubscribe(func() {})
	server.broadcaster.OnUnsubscribe(func() {})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.startProducer(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", server.eventsHandler)

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	cancel1 := connectSSEClient(t, httpServer.URL+"/events")
	defer cancel1()

	time.Sleep(200 * time.Millisecond)

	if count := server.broadcaster.SubscriberCount(); count != 1 {
		t.Errorf("expected 1 subscriber after connect, got %d", count)
	}

	cancel2 := connectSSEClient(t, httpServer.URL+"/events")
	defer cancel2()

	time.Sleep(200 * time.Millisecond)

	if count := server.broadcaster.SubscriberCount(); count != 2 {
		t.Errorf("expected 2 subscribers after second connect, got %d", count)
	}
}

// --- Graceful shutdown test ---

func TestGracefulShutdown(t *testing.T) {
	t.Parallel()

	server := newActivityServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.startProducer(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", server.eventsHandler)

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	stop := connectSSEClient(t, httpServer.URL+"/events")
	defer stop()

	time.Sleep(200 * time.Millisecond)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	cancel()

	if err := server.broadcaster.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("broadcaster shutdown error: %v", err)
	}

	if health := server.broadcaster.Health(); !health.Closed {
		t.Error("expected broadcaster to be closed after shutdown")
	}
}

// --- Helpers ---

func makeTestEvent(id int64) sse.Event {
	return sse.Event{
		Event: "datastar-patch-elements",
		Data:  "selector #feed\nmode prepend\ncategory info\nelements <div>test</div>",
		ID:    sse.NewEventID(strconv.FormatInt(id, 10)),
	}
}

// collectSSEEvents connects to an SSE endpoint, reads events for the given
// duration, and returns the raw event blocks as strings.
func collectSSEEvents(url string, duration time.Duration) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), duration+2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connect to SSE endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var events []string

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var current strings.Builder

	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			if current.Len() > 0 {
				events = append(events, current.String())
			}

			return events, nil
		default:
		}

		if !scanner.Scan() {
			if current.Len() > 0 {
				events = append(events, current.String())
			}

			// A non-nil scanner error means a genuine read failure (e.g. a line
			// exceeding the buffer, or a dropped connection). A context
			// deadline/cancellation from the bounded collection window is the
			// normal stop and must not fail the test.
			if scanErr := scanner.Err(); scanErr != nil &&
				!errors.Is(scanErr, context.Canceled) &&
				!errors.Is(scanErr, context.DeadlineExceeded) {
				return events, fmt.Errorf("scan SSE stream: %w", scanErr)
			}

			return events, nil
		}

		line := scanner.Text()

		if line == "" {
			if current.Len() > 0 {
				events = append(events, current.String())
				current.Reset()
			}

			continue
		}

		current.WriteString(line)
		current.WriteString("\n")
	}
}

// extractEventIDs pulls the id: field values from SSE event blocks.
func extractEventIDs(events []string) []string {
	var ids []string

	for _, evt := range events {
		for _, line := range strings.Split(evt, "\n") {
			if strings.HasPrefix(line, "id:") {
				id := strings.TrimSpace(strings.TrimPrefix(line, "id:"))
				if id != "" {
					ids = append(ids, id)
				}
			}
		}
	}

	return ids
}

// commonIDs returns IDs present in both slices.
func commonIDs(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, id := range b {
		set[id] = true
	}

	var result []string

	for _, id := range a {
		if set[id] {
			result = append(result, id)
		}
	}

	return result
}

// connectSSEClient opens an SSE connection and returns a cancel function
// to close it. Used for subscriber-count and shutdown tests.
func connectSSEClient(t *testing.T, url string) context.CancelFunc {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	req.Header.Set("Accept", "text/event-stream")

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}

		defer func() { _ = resp.Body.Close() }()

		_, _ = io.Copy(io.Discard, resp.Body)
	}()

	return func() {
		cancel()
		wg.Wait()
	}
}
