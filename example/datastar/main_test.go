package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

// --- memStore unit tests ---

// --- Concurrent fan-out test ---

func TestConcurrentFanOut(t *testing.T) {
	t.Parallel()

	server := newActivityServer()
	defer server.broadcaster.Close()

	ctx := t.Context()

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

	ctx := t.Context()

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
				sse.NewEventID("1"),
				activityItem{
					category: categoryAlert,
					badge:    badgeAlert,
					message:  "test alert",
					time:     "00:00:01",
				},
			),
			feedItemEvent(
				sse.NewEventID("2"),
				activityItem{
					category: categorySuccess,
					badge:    badgeSuccess,
					message:  "test ok",
					time:     "00:00:02",
				},
			),
			feedItemEvent(
				sse.NewEventID("3"),
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

	ctx := t.Context()

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

// --- listenAddr unit tests (serial: t.Setenv cannot coexist with t.Parallel) ---

func TestListenAddr_Default(t *testing.T) {
	t.Setenv("PORT", "")

	if got, want := listenAddr(), ":"+defaultDatastarPort; got != want {
		t.Errorf("listenAddr() = %q, want %q", got, want)
	}
}

func TestListenAddr_PortOverride(t *testing.T) {
	t.Setenv("PORT", "18765")

	if got, want := listenAddr(), ":18765"; got != want {
		t.Errorf("listenAddr() = %q, want %q", got, want)
	}
}
