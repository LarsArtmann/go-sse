package ssetest_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/larsartmann/go-sse"
	"github.com/larsartmann/go-sse/ssetest"
)

func TestCollect_WithPath(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		_ = stream.Send(sse.Event{Event: "feed", Data: "from /events"})
	})

	events := ssetest.Collect(t, mux, ssetest.WithPath("/events?filter=alerts"))
	ssetest.RequireEventCount(t, events, 1)
	ssetest.RequireData(t, events[0], "from /events")

	if got := events[0].Data(); got != "from /events" {
		t.Errorf("data: got %q", got)
	}
}

func TestCollect_WithHeader(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test-Trace"); got != "abc123" {
			http.Error(w, "missing trace header", http.StatusBadRequest)

			return
		}

		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		_ = stream.Send(sse.Event{Event: "feed", Data: "ok"})
	})

	events := ssetest.Collect(t, handler,
		ssetest.WithHeader("X-Test-Trace", "abc123"),
		ssetest.WithHeader("X-Test-Trace", "duplicate-appends"),
	)
	ssetest.RequireEventCount(t, events, 1)
}

func TestCollect_WithLastEventID_HeaderArrives(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastID := r.Header.Get("Last-Event-ID")

		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		_ = stream.Send(sse.Event{Event: "lastID", Data: lastID})
	})

	events := ssetest.Collect(t, handler, ssetest.WithLastEventID("42"))
	ssetest.RequireEventCount(t, events, 1)
	ssetest.RequireData(t, events[0], "42")
}

func TestCollectN_WithOptions(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Last-Event-ID"); got != "1" {
			http.Error(w, "missing Last-Event-ID", http.StatusBadRequest)

			return
		}

		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		for i := range 5 {
			_ = stream.Send(sse.Event{Event: "feed", Data: strconv.Itoa(i)})
		}

		<-stream.Context().Done()
	})

	events := ssetest.CollectN(t, mux, 2,
		ssetest.WithPath("/stream"),
		ssetest.WithLastEventID("1"),
	)
	ssetest.RequireEventCount(t, events, 2)
}
