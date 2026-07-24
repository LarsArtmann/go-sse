# go-sse

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-sse.svg)](https://pkg.go.dev/github.com/larsartmann/go-sse)

Server-Sent Events transport for Go — wire format, connection management, fan-out broadcasting, and reconnection replay. Two small dependencies (`go-branded-id`, `go-error-family`), no framework or payload-format opinions.

## Why

Every Go project that serves SSE reinvents the same four pieces: event serialization, connection lifecycle, subscriber fan-out, and reconnection replay. This library is those four pieces, extracted from production use, with no opinions about your domain, your framework, or your payload format.

## What's Included

| Component               | What it does                                                             |
| ----------------------- | ------------------------------------------------------------------------ |
| `Event`, `EventID`      | SSE wire-format types — event name, data, id, retry                      |
| `WriteEvent`            | Allocation-minimized SSE serializer (handles multi-line data, CRLF)      |
| `Stream`                | Single SSE connection — headers, send, heartbeat, context, Last-Event-ID |
| `Broadcaster[T]`        | Generic subscriber fan-out — subscribe, broadcast, close, hooks          |
| `EventStore` + `Replay` | Reconnection replay — missed events sent on reconnect                    |

## What's NOT Included

- No CQRS dispatch hooks
- No dashboard server, no routes, no HTML templates
- No WebSocket support
- No event bus integration
- No opinion about your payload format (strings, JSON, HTML fragments — all fine)

## Install

```bash
go get github.com/larsartmann/go-sse
```

Requires Go 1.26+ with `GOEXPERIMENT=jsonv2` (transitive dependency on go-branded-id).

## Quick Start

### SSE endpoint with live broadcasting

```go
broadcaster := sse.NewBroadcaster[sse.Event]()

// SSE endpoint
mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
    stream := sse.NewStream(w, r)
    defer stream.Close()

    ch := broadcaster.Subscribe()
    defer broadcaster.Unsubscribe(ch)

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
})

// Push from anywhere — all connected clients receive it
broadcaster.Broadcast(sse.Event{Event: "update", Data: "<div>new</div>"})
```

### Heartbeat to keep connections alive through proxies

```go
stream := sse.NewStream(w, r)
defer stream.Close()

// Prevents Nginx/Cloudflare/AWS ALB from killing idle connections
go stream.Heartbeat(stream.Context(), 15*time.Second)
```

### Reconnection replay

When a browser reconnects after a drop, it sends the `Last-Event-ID` header.

```go
stream := sse.NewStream(w, r)
defer stream.Close()

// Replay missed events
if lastID := stream.LastEventID(); !lastID.IsZero() {
    sse.Replay(stream, store, lastID)
}

// Then continue with live events...
```

Implement the `EventStore` interface with whatever backing store you use (in-memory, database, event journal):

```go
type YourStore struct{}

func (s *YourStore) EventsAfter(lastID sse.EventID) []sse.Event {
    // Return events with IDs strictly after lastID, ordered ascending
}
```

### Event ID validation

```go
// Safe construction from trusted input
id := sse.NewEventID("evt-42")

// Validation from untrusted input (rejects newlines that corrupt the wire format)
id, err := sse.ParseEventID(headerValue)
```

## API Reference

### Event Types

```go
type Event struct {
    Event string   // event name (maps to event: field)
    Data  string   // payload (multi-line data splits into data: per line)
    ID    EventID  // event identifier (maps to id: field)
    Retry uint     // reconnection interval in milliseconds (0 = omit)
}

type EventID = brandid.ID[eventBrand, string] // branded — prevents cross-assignment
```

### Writing Events

```go
sse.WriteEvent(w, sse.Event{Event: "update", Data: "<div>new</div>"})
sse.WriteHeartbeat(w)       // ": heartbeat\n\n"
sse.WriteRetry(w, 5000)     // "retry: 5000\n\n"

// Event.String() — compact debug representation (NOT the wire format)
log.Printf("dropped event: %s", evt)
// Output: event=update id=42 data=<div>new</div>
```

### Stream (single connection)

```go
stream := sse.NewStream(w, r)     // sets headers + 200 OK
defer stream.Close()

stream.Send(sse.Event{...})        // write + flush
stream.SendHTML("update", "<div>") // convenience: send raw HTML fragment
stream.SendJSON("update", payload) // convenience: marshal JSON, send as data
stream.Heartbeat(ctx, 15*time.Second)
stream.LastEventID()               // Last-Event-ID header
stream.Context()                   // cancelled on disconnect
stream.OnDisconnect(func() { ... })
```

### Broadcaster (fan-out)

```go
b := sse.NewBroadcaster[sse.Event]()

ch := b.Subscribe()                 // returns <-chan sse.Event
defer b.Unsubscribe(ch)             // closes the channel

b.Broadcast(sse.Event{...})         // non-blocking, drops to slow consumers
b.BroadcastMany(evt1, evt2, evt3)   // batch: single lock pass, preserves order
b.SubscriberCount()
b.Close()                           // closes all subscriber channels

b.OnSubscribe(func() { ... })       // connection callback
b.OnUnsubscribe(func() { ... })     // disconnection callback
```

`Broadcaster` is generic — use `sse.NewBroadcaster[sse.Event]()` for standard SSE events, or `sse.NewBroadcaster[string]()` for raw string messages (useful for WebSocket fan-out).

### Replay (reconnection)

```go
type EventStore interface {
    EventsAfter(lastID EventID) []Event
}

n, err := sse.Replay(stream, store, lastEventID)
```

## Design Decisions

- **Non-blocking broadcast**: Slow consumers never block the broadcaster. Events are dropped when a subscriber's 64-deep buffer is full. This prevents head-of-line blocking. Consumers recover via snapshot/replay on reconnect.
- **Mutex-protected Stream**: `Send` and `Heartbeat` serialize on a mutex because `http.ResponseWriter` is not safe for concurrent use. Both goroutines can write safely.
- **Channel pointer identity**: `Unsubscribe` uses `reflect.ValueOf(ch).Pointer()` for O(1) lookup. No subscriber IDs to manage.
- **Branded EventID**: `EventID` uses `go-branded-id` to prevent accidental cross-assignment with other string-typed IDs in your codebase.
- **Zero allocation on fast path**: `WriteEvent` uses direct byte appends. No `fmt.Fprintf` on the SSE hot path.

## Non-Blocking Drop Policy

`Broadcaster.Broadcast` and `BroadcastMany` never block. Each subscriber has a
64-message buffer; if it's full, new events are silently dropped for that
subscriber. This prevents one slow consumer from stalling the entire fan-out.
`BroadcastMany` acquires the read lock once for the entire batch, guaranteeing
per-subscriber ordering across the batch.

**Implications:**

- A subscriber processing events slower than they arrive will lose messages.
- There is no per-message acknowledgment or retry in the transport layer.
- Consumers needing guaranteed delivery must implement application-level
  reconnection with `Last-Event-ID` + `Replay` to recover missed events.
- The drop is per-subscriber: fast consumers are unaffected by slow ones.

## License

MIT
