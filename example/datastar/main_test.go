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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			events, err := store.EventsAfter(sse.NewEventID(testCase.idStr))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(events) != testCase.want {
				t.Errorf(
					"id=%q: expected %d events, got %d",
					testCase.idStr,
					testCase.want,
					len(events),
				)
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

	const collectDuration = 3 * time.Second

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
		t.Fatal("client A received 0 events: fan-out or producer is broken")
	}

	if len(eventsB) == 0 {
		t.Fatal("client B received 0 events: fan-out or producer is broken")
	}

	idsA := extractEventIDs(eventsA)
	idsB := extractEventIDs(eventsB)

	if len(idsA) == 0 {
		t.Fatal("client A received 0 events with IDs; expected feed-item events")
	}

	common := commonIDs(idsA, idsB)
	if len(common) == 0 {
		t.Fatalf(
			"no common event IDs between clients A and B; fan-out is broken\nA: %v\nB: %v",
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

// --- Filter predicate test ---

func TestFilterPredicate_AlertsOnly(t *testing.T) {
	t.Parallel()

	server := newActivityServer()
	defer server.broadcaster.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", server.eventsHandler)

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	// Broadcast known events after a short delay so the subscriber
	// is connected first. One alert, one success, one info ; only
	// the alert should pass the ?filter=alerts predicate.
	go func() {
		time.Sleep(200 * time.Millisecond)

		server.broadcaster.BroadcastMany(
			feedItemEvent(
				1,
				activityItem{
					category: categoryAlert,
					badge:    badgeAlert,
					message:  "test alert",
					time:     "00:00:01",
				},
			),
			feedItemEvent(
				2,
				activityItem{
					category: categorySuccess,
					badge:    badgeSuccess,
					message:  "test ok",
					time:     "00:00:02",
				},
			),
			feedItemEvent(
				3,
				activityItem{
					category: categoryInfo,
					badge:    badgeInfo,
					message:  "test info",
					time:     "00:00:03",
				},
			),
		)
	}()

	events, err := collectSSEEvents(httpServer.URL+"/events?filter=alerts", 2*time.Second)
	if err != nil {
		t.Fatalf("collect filtered events: %v", err)
	}

	var feedItems int

	for _, evt := range events {
		if !strings.Contains(evt, "id:") {
			continue // meta event (no ID) ; filter always passes these
		}

		feedItems++

		if !strings.Contains(evt, "category alert") {
			t.Errorf("non-alert feed item passed through alerts-only filter:\\n%s", evt)
		}

		if strings.Contains(evt, "category success") || strings.Contains(evt, "category info") {
			t.Errorf("non-alert category leaked through alerts-only filter:\\n%s", evt)
		}
	}

	if feedItems != 1 {
		t.Fatalf("expected exactly 1 alert feed item through filter, got %d", feedItems)
	}

	t.Log("filter verified: 1 alert feed item passed, 0 non-alert leaks")
}

// --- CORS header test ---

func TestEventsHandler_CORSHeader(t *testing.T) {
	t.Parallel()

	server := newActivityServer()
	defer server.broadcaster.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", server.eventsHandler)

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpServer.URL+"/events", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect to /events: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin: expected %q, got %q", "*", got)
	}
}

// --- $replayed reset on subscribe ---

func TestReplayedSignalResetOnSubscribe(t *testing.T) {
	t.Parallel()

	server := newActivityServer()
	defer server.broadcaster.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.startProducer(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", server.eventsHandler)

	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	// OnSubscribe broadcasts replayEvent(0) to clear the replay banner.
	// It fires immediately when our subscriber connects, so it arrives
	// among the first events, well within a short collection window.
	events, err := collectSSEEvents(httpServer.URL+"/events", 2*time.Second)
	if err != nil {
		t.Fatalf("collect events: %v", err)
	}

	for _, evt := range events {
		if strings.Contains(evt, `"replayed":0`) {
			t.Log("replayed reset verified: OnSubscribe broadcasts replayEvent(0)")

			return
		}
	}

	t.Fatal("no replayed:0 signal found; OnSubscribe replayEvent(0) not firing")
}

// --- Helpers ---

func makeTestEvent(id int64) sse.Event {
	return sse.Event{
		Event: eventPatchElements,
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

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Read lines on a goroutine so the timer fires even when no new
	// data is arriving on the stream (the old busy-wait pattern
	// blocked indefinitely inside scanner.Scan between events).
	type scanLine struct {
		text string
		ok   bool
	}

	lineCh := make(chan scanLine)

	go func() {
		for scanner.Scan() {
			lineCh <- scanLine{text: scanner.Text(), ok: true}
		}
		lineCh <- scanLine{ok: false}
	}()

	timer := time.NewTimer(duration)
	defer timer.Stop()

	var (
		events  []string
		current strings.Builder
	)

	for {
		select {
		case <-timer.C:
			if current.Len() > 0 {
				events = append(events, current.String())
			}

			return events, nil
		case line := <-lineCh:
			if !line.ok {
				if current.Len() > 0 {
					events = append(events, current.String())
				}
				if scanErr := scanner.Err(); scanErr != nil &&
					!errors.Is(scanErr, context.Canceled) &&
					!errors.Is(scanErr, context.DeadlineExceeded) {
					return events, fmt.Errorf("scan SSE stream: %w", scanErr)
				}

				return events, nil
			}
			if line.text == "" {
				if current.Len() > 0 {
					events = append(events, current.String())
					current.Reset()
				}
			} else {
				current.WriteString(line.text)
				current.WriteString("\n")
			}
		}
	}
}

// extractEventIDs pulls the id: field values from SSE event blocks.
func extractEventIDs(events []string) []string {
	var ids []string

	for _, evt := range events {
		for line := range strings.SplitSeq(evt, "\n") {
			if rest, ok := strings.CutPrefix(line, "id:"); ok {
				id := strings.TrimSpace(rest)
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
