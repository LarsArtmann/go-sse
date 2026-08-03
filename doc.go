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
// # DataStar Retry and Reconnection
//
// DataStar has two independent retry layers:
//
//  1. SSE protocol retry: the "retry:" field (set via [Event.Retry] or
//     [WriteRetry]) tells the browser's EventSource how long to wait before
//     reconnecting after a connection drop. go-sse supports this through
//     [Event.Retry] and [WriteRetry].
//
//  2. DataStar client retry: the @get() action's retry, retryInterval,
//     retryScaler, retryMaxWait, retryMaxCount options control automatic
//     re-execution of the fetch action when the request itself fails (network
//     errors, 4xx/5xx responses).
//
// These layers are independent. For most use cases:
//
//   - Set [Event.Retry] to control browser-level reconnection timing.
//   - Use [Stream.LastEventID] and [Replay] to recover missed events on
//     reconnection (go-sse handles the SSE protocol side).
//   - Let DataStar's client-side retry options handle transient fetch failures
//     (the browser will re-issue the @get() automatically).
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
// # Filtered Subscriptions
//
// For applications that need selective delivery (e.g., per-channel or per-guild
// event routing), use [Broadcaster.SubscribeFilter] with a predicate function.
// The predicate is checked before the channel send, so irrelevant events never
// enter the subscriber's buffer:
//
//	// Deliver only message events to this subscriber
//	ch := broadcaster.SubscribeFilter(func(evt sse.Event) bool {
//	    return evt.Event == "message"
//	})
//	defer broadcaster.Unsubscribe(ch)
//
// The predicate is called inside the fan-out loop under the read lock — it must
// be pure, fast, and non-blocking. A panicking predicate is recovered and
// treated as a non-match (the event is skipped for that subscriber); one broken
// predicate cannot crash the broadcaster.
//
// For filtered reconnection replay, use [ReplayFiltered] with a
// [FilteredEventStore]. The predicate is pushed into the store query so the
// replay budget is spent entirely on matching events:
//
//	sse.ReplayFiltered(stream, store, lastID, func(evt sse.Event) bool {
//	    return evt.Event == "message"
//	})
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
