// Package main implements a minimal SSE server using go-sse.
//
// Run: go run example/server.go
// Then: curl -N http://localhost:8080/events
// Push: curl -X POST http://localhost:8080/broadcast?msg=hello
package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/larsartmann/go-sse"
)

const (
	addr              = ":8080"
	heartbeatInterval = 15 * time.Second
)

func main() {
	bc := sse.NewBroadcaster[sse.Event]()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", eventsHandler(bc))
	mux.HandleFunc("POST /broadcast", broadcastHandler(bc))

	log.Printf("SSE server on http://localhost%s", addr)
	log.Fatal(
		http.ListenAndServe(
			addr,
			mux,
		),
	)
}

func eventsHandler(bc *sse.Broadcaster[sse.Event]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		if lastID := stream.LastEventID(); !lastID.IsZero() {
			log.Printf("client reconnecting from %s", lastID.Get()) //nolint:gosec // example
		}

		_ = stream.Send(sse.Event{
			Event: sse.EventConnected,
			Data:  "connected",
			ID:    sse.NewEventID("0"),
		})

		ch := bc.Subscribe()
		defer bc.Unsubscribe(ch)

		go stream.Heartbeat(stream.Context(), heartbeatInterval)

		for {
			select {
			case <-stream.Context().Done():
				return
			case evt, ok := <-ch:
				if !ok || stream.Send(evt) != nil {
					return
				}
			}
		}
	}
}

func broadcastHandler(bc *sse.Broadcaster[sse.Event]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg := r.URL.Query().Get("msg")
		if msg == "" {
			http.Error(w, "missing msg query parameter", http.StatusBadRequest)

			return
		}

		bc.Broadcast(sse.Event{
			Event: "message",
			Data:  msg,
			ID:    sse.NewEventID(strconv.FormatInt(time.Now().UnixNano(), 10)),
		})

		w.WriteHeader(http.StatusAccepted)
	}
}
