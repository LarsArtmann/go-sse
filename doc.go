// Package sse provides Server-Sent Events (SSE) transport primitives for Go.
//
// This package implements the SSE wire format ([Event], [EventID],
// [WriteEvent]), connection management ([Stream]), subscriber fan-out
// ([Broadcaster]), and reconnection replay ([EventStore], [Replay]).
//
// It has zero knowledge of application domains, CQRS dispatch, or dashboard
// rendering. Consumers build domain-specific layers on top.
//
// # Quick Start
//
//	Create a [Broadcaster], wire SSE endpoints, and push events:
//
//	broadcaster := sse.NewBroadcaster[sse.Event]()
//
//	// SSE endpoint
//	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
//	    stream := sse.NewStream(w, r)
//	    defer stream.Close()
//
//	    ch := broadcaster.Subscribe()
//	    defer broadcaster.Unsubscribe(ch)
//
//	    for {
//	        select {
//	        case <-stream.Context().Done():
//	            return
//	        case evt, ok := <-ch:
//	            if !ok || stream.Send(evt) != nil {
//	                return
//	            }
//	        }
//	    }
//	})
//
//	// Push from anywhere
//	broadcaster.Broadcast(sse.Event{Event: "update", Data: "<div>new</div>"})
//
// # SSE Wire Format
//
//	[WriteEvent] serializes an [Event] into the standard SSE wire format:
//
//	event: update
//	data: <div>new</div>
//
//	(terminated by a blank line)
//
// Multi-line data is split so each line gets its own "data:" prefix, as
// required by the [SSE specification].
//
// # Reconnection
//
// When a browser reconnects after a drop, it sends the Last-Event-ID header
// containing the last event ID it received. Use [Stream.LastEventID] to read it,
// then [Replay] with an [EventStore] to send missed events:
//
//	stream := sse.NewStream(w, r)
//	if lastID := stream.LastEventID(); !lastID.IsZero() {
//	    sse.Replay(stream, store, lastID)
//	}
//
// [SSE specification]: https://html.spec.whatwg.org/multipage/server-sent-events.html
package sse
