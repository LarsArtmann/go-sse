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
//	    defer func() { _ = stream.Close() }()
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
// # Keyed Data Lines (DataStar)
//
// Some SSE protocols — most notably [DataStar] — use keyed data lines, where
// each "data:" line carries a "key value" pair and multi-line values repeat
// the key prefix:
//
//	event: datastar-patch-elements
//	data: selector #feed
//	data: mode inner
//	data: elements <div>
//	data: elements   <span>1</span>
//	data: elements </div>
//
// [KeyedLines] builds the prefixed string for multi-line values; [Stream.SendLines]
// sends an event from multiple data-line arguments. Together they produce the
// wire format above:
//
//	stream.SendLines("datastar-patch-elements",
//	    "selector #feed",
//	    "mode inner",
//	    sse.KeyedLines("elements", html),
//	)
//
// For single-key events (e.g., patch-signals), use the convenience helpers:
//
//	stream.SendKeyed("datastar-patch-signals", "signals", `{"progress":50}`)
//	sse.WriteKeyedLines(w, "datastar-patch-signals", "signals", `{"progress":50}`)
//
// WriteKeyedLines is the wire-only variant (no net/http) for consumers that
// manage their own HTTP scaffolding around WriteEvent.
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
// # Concurrency and Memory Model
//
// Each subscriber gets an independent buffered channel (capacity 64).
// [Broadcaster.Broadcast] and [Broadcaster.BroadcastMany] are non-blocking: if a
// subscriber's buffer is full, the event is silently dropped for that
// subscriber only — fast consumers are never stalled by slow ones.
//
// [Broadcaster.BroadcastMany] acquires the read lock once for the entire batch,
// guaranteeing per-subscriber ordering across the batch.
//
// [Stream.Send] and [Stream.Heartbeat] serialize on a mutex because
// http.ResponseWriter is not safe for concurrent use. Both goroutines can write
// safely. [Stream.Close] is safe to call concurrently with Send and Heartbeat.
//
// Consumers needing guaranteed delivery must implement application-level
// reconnection with Last-Event-ID + [Replay].
//
// [SSE specification]: https://html.spec.whatwg.org/multipage/server-sent-events.html
// [DataStar]: https://data-star.dev
package sse
