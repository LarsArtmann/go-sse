# [go-sse](https://pkg.go.dev/github.com/larsartmann/go-sse)

[![CI](https://github.com/larsartmann/go-sse/actions/workflows/ci.yml/badge.svg)](https://github.com/larsartmann/go-sse/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-sse.svg)](https://pkg.go.dev/github.com/larsartmann/go-sse)
[![Release](https://img.shields.io/github/v/release/larsartmann/go-sse.svg)](https://github.com/larsartmann/go-sse/releases)

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
| `SubscribeFilter`       | Predicate-based subscription — only matching events enter the buffer     |
| `EventStore` + `Replay` | Reconnection replay — missed events sent on reconnect                    |
| `FilteredEventStore`    | Predicate push-down for replay budget efficiency                         |

## What's NOT Included

- No CQRS dispatch hooks
- No dashboard server, no routes, no HTML templates
- No WebSocket support
- No event bus integration
- No opinion about your payload format (strings, JSON, HTML fragments — all fine)
- No `Broadcaster.ServeSSE` convenience handler (would bake in heartbeat, replay, and event-loop opinions; the `example/` package shows the canonical pattern)

## Install

```bash
go get github.com/larsartmann/go-sse
```

Requires Go 1.26.7+ (see `go.mod`) with `GOEXPERIMENT=jsonv2` (transitive dependency on go-branded-id).

## Quick Start

### SSE endpoint with live broadcasting

```go
broadcaster := sse.NewBroadcaster[sse.Event]()

// SSE endpoint
mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
    stream := sse.NewStream(w, r)
    defer func() { _ = stream.Close() }()

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
defer func() { _ = stream.Close() }()

// Prevents Nginx/Cloudflare/AWS ALB from killing idle connections
go stream.Heartbeat(stream.Context(), 15*time.Second)
```

### Reconnection replay

When a browser reconnects after a drop, it sends the `Last-Event-ID` header.

```go
stream := sse.NewStream(w, r)
defer func() { _ = stream.Close() }()

// Replay missed events
if lastID := stream.LastEventID(); !lastID.IsZero() {
    if _, err := sse.Replay(stream, store, lastID); err != nil {
        log.Printf("replay failed: %v", err)
    }
}

// Then continue with live events...
```

Implement the `EventStore` interface with whatever backing store you use (in-memory, database, event journal):

```go
type YourStore struct{}

func (s *YourStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
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

### Filtered subscriptions

Deliver only events matching a predicate to a subscriber — useful for per-channel,
per-tenant, or per-event-type routing without multiple broadcasters:

```go
b := sse.NewBroadcaster[sse.Event]()

// Only "message" events enter this subscriber's buffer
ch := b.SubscribeFilter(func(evt sse.Event) bool {
    return evt.Event == "message"
})
defer b.Unsubscribe(ch)
```

The predicate is checked before the non-blocking send. It runs inside the fan-out
loop under the read lock — it must be pure, fast, and non-blocking.

For filtered reconnection replay, use `ReplayFiltered`. If the store implements
`FilteredEventStore`, the predicate is pushed into the store query so the replay
budget is spent entirely on matching events:

```go
sse.ReplayFiltered(stream, store, lastID, func(evt sse.Event) bool {
    return evt.Event == "message"
})
```

## Testing Your Handlers

[`ssetest`](./ssetest/) is a companion module with deep-testing helpers: it spins up a real HTTP server, drives your handler, parses the SSE wire format, and hands back typed, assertable events. No hand-rolled parser required.

```go
import "github.com/larsartmann/go-sse/ssetest"

func TestFeedHandler(t *testing.T) {
    t.Parallel()

    events := ssetest.Collect(t, myHandler)
    ssetest.RequireEventCount(t, events, 2)
    ssetest.RequireEventType(t, events[0], "feed")
    ssetest.RequireData(t, events[0], "hello")

    // Simulate a reconnecting browser for replay testing:
    replayed := ssetest.Collect(t, myHandler, ssetest.WithLastEventID("42"))
}
```

`CollectN` handles streaming handlers (reads exactly N events), `CollectWithTimeout` returns whatever arrived before a deadline, and `ReadEvents`/`ReadNEvents` parse any `io.Reader`. For test patterns that interleave reads with actions (POST, mutate state, read the next event), use a [`StreamReader`](./ssetest/README.md) — it keeps a single scanner across `Next()` calls so buffered data is never lost. Per the SSE spec, dataless frames (heartbeats, id-only) never surface as events. All helpers accept `testing.TB`, so they work with `*testing.T`, `*testing.B`, and Ginkgo's `GinkgoT()`. See [ssetest/README.md](./ssetest/README.md).

The ssetest parser is spec-conformant to WHATWG HTML § 9.2.6 and pinned by the official Web Platform Tests `eventsource/format-*` corpus, re-run through 1–4096-byte chunked readers, and closed against `WriteEvent` with round-trip property tests (`ssetest/wpt_format_corpus_test.go`, `chunk_boundary_test.go`, `roundtrip_test.go`).

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

sse.KeyedLines("elements", html) // prefix each line with "elements " (DataStar pattern)
sse.JoinLines("selector #feed", "mode inner", sse.KeyedLines("elements", html)) // multi-line Event.Data
sse.WriteKeyedLines(w, "event", "key", "value") // wire-only: WriteEvent + KeyedLines

// Event.String() — compact debug representation (NOT the wire format)
log.Printf("dropped event: %s", evt)
// Output: {event:update id:42 retry:3000 data:<div>new</div>}
```

### Stream (single connection)

```go
stream := sse.NewStream(w, r)     // sets headers + 200 OK
defer func() { _ = stream.Close() }()

stream.Send(sse.Event{...})        // write + flush
stream.SendData("update", "<div>") // convenience: send raw string data
stream.SendJSON("update", payload) // convenience: marshal JSON, send as data
stream.SendLines("event", "k1 v1", "k2 v2") // convenience: multi-line data lines
stream.SendKeyed("event", "key", "value")      // convenience: single-key DataStar pattern
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

// Predicate-based: only matching events enter the buffer
ch := b.SubscribeFilter(func(evt sse.Event) bool { return evt.Event == "message" })

b.Broadcast(sse.Event{...})         // non-blocking, drops to slow consumers
b.BroadcastMany(evt1, evt2, evt3)   // batch: single lock pass, preserves order
b.SubscriberCount()
b.Close()                           // closes all subscriber channels

b.OnSubscribe(func() { ... })       // connection callback
b.OnUnsubscribe(func() { ... })     // disconnection callback
```

`Broadcaster` is generic — use `sse.NewBroadcaster[sse.Event]()` for standard SSE events, or `sse.NewBroadcaster[YourType]()` to fan out any message type.

### Replay (reconnection)

```go
type EventStore interface {
    EventsAfter(lastID EventID) ([]Event, error)
}

// Stores that can push predicates into queries implement this additionally:
type FilteredEventStore interface {
    EventStore
    EventsAfterFiltered(lastID EventID, pred func(Event) bool) ([]Event, error)
}

n, err := sse.Replay(stream, store, lastEventID)
n, err := sse.ReplayFiltered(stream, store, lastEventID, pred) // filtered replay
```

## Design Decisions

- **Non-blocking broadcast**: Slow consumers never block the broadcaster. Events are dropped when a subscriber's 64-deep buffer is full. This prevents head-of-line blocking. Consumers recover via snapshot/replay on reconnect.
- **Mutex-protected Stream**: `Send` and `Heartbeat` serialize on a mutex because `http.ResponseWriter` is not safe for concurrent use. Both goroutines can write safely.
- **Channel pointer identity**: `Unsubscribe` uses `reflect.ValueOf(ch).Pointer()` for O(1) lookup. No subscriber IDs to manage.
- **Branded EventID**: `EventID` uses `go-branded-id` to prevent accidental cross-assignment with other string-typed IDs in your codebase.
- **Zero allocation on fast path**: `WriteEvent` uses direct byte appends. No `fmt.Fprintf` on the SSE hot path.
- **Predicate under read lock**: `SubscribeFilter` predicates run inside the fan-out loop under the read lock. This is intentional — the predicate must be pure, fast, and non-blocking. A panicking predicate is recovered and treated as a non-match (the event is skipped for that subscriber). One broken predicate cannot crash the broadcaster.

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

## Framework Integration: DataStar

[DataStar](https://data-star.dev) is a hypermedia library that uses SSE as its
primary transport. go-sse supports it out of the box — the `KeyedLines` helper
and `SendLines` method handle DataStar's keyed-data-line wire format, where
multi-line values repeat the key prefix on each line.

### Serving DataStar SSE events

```go
mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
    stream := sse.NewStream(w, r)
    defer func() { _ = stream.Close() }()

    // Patch elements: morph a feed div with multi-line HTML
    html := `<div id="feed">
  <span>1</span>
</div>`

    _ = stream.SendLines("datastar-patch-elements",
        "selector #feed",
        "mode inner",
        sse.KeyedLines("elements", html),
    )

    // Patch signals: update client-side reactive state
    _ = stream.SendLines("datastar-patch-signals",
        sse.KeyedLines("signals", `{"progress": 50}`),
    )
})
```

### What go-sse provides for DataStar

| DataStar requirement          | go-sse support                                                             |
| ----------------------------- | -------------------------------------------------------------------------- |
| `text/event-stream` + headers | `NewStream` sets all required headers automatically                        |
| `event: datastar-*` field     | `Event.Event`                                                              |
| `data: key value` multi-line  | `JoinLines` + `KeyedLines` + `SendLines` + `WriteKeyedLines` + `SendKeyed` |
| `id:` / `retry:` fields       | `Event.ID` / `Event.Retry`                                                 |
| `Last-Event-ID` reconnection  | `Stream.LastEventID()` + `Replay`                                          |
| Heartbeat (proxy keep-alive)  | `Stream.Heartbeat`                                                         |

go-sse has no DataStar-specific types or event-name constants — it remains a
transport library. `KeyedLines` is a general SSE utility (keyed data lines are
used by many protocols); DataStar is the most prominent consumer.

### Runnable example: live activity feed

A complete DataStar example lives in [`example/datastar/`](example/datastar/); a minimal HTMX SSE example lives in [`example/htmx/`](example/htmx/). [`example/README.md`](example/README.md) compares the two approaches (mechanism, payload size, granularity, bundle size, trade-offs).
It is a **live activity feed** that demonstrates every go-sse feature in under
30 seconds of interaction.

| You do this...               | ...and see this go-sse feature                                                  |
| ---------------------------- | ------------------------------------------------------------------------------- |
| Open multiple browser tabs   | All tabs receive the same events live — `Broadcaster` fan-out                   |
| Watch the subscriber counter | Updates in real time — `SubscriberCount` via `OnSubscribe`/`OnUnsubscribe`      |
| Click "Alerts only"          | Feed instantly filters — `SubscribeFilter` with a predicate                     |
| Close a tab, wait, reopen it | "Replayed N missed events" banner — `EventStore` + `Replay` via `Last-Event-ID` |
| Leave a tab idle             | Connection stays alive — `Heartbeat`                                            |

The example uses [templ](https://templ.guide) for type-safe HTML, a real `.css`
file with dark/light theme support, and embeds the DataStar JS bundle via
`go:embed` — no CDN required.

```bash
go run ./example/datastar/
# open http://localhost:8765
```

### Try It (60-second checklist)

1. **Fan-out** — Open `http://localhost:8765` in 2+ browser tabs. All tabs show the same events arriving simultaneously. The "clients connected" counter updates in real time.
2. **Filtering** — Click "Alerts only" in one tab. Only ALERT events appear. Other tabs stay unfiltered. Press `a` / `e` as keyboard shortcuts.
3. **Reconnection replay** — Close a tab, wait 6+ seconds (3+ missed events), reopen it. The "Replayed N missed events" banner appears briefly with the gap filled.
4. **Heartbeat** — Leave a tab idle for 60+ seconds. The connection stays alive through the green pulsing "live" indicator.
5. **Shutdown** — Press `Ctrl+C` in the terminal. Connected clients are drained gracefully within 5 seconds.

### How It Works

**Data flow:**

```
background producer goroutine
    │  generates a random activity item every 2s
    │  assigns a sequential event ID
    ▼
memStore.Append(evt) ──── stores last 50 events for replay
    │
    ▼
Broadcaster.BroadcastMany(evt, totalSignal)
    │  non-blocking fan-out: each subscriber gets the event
    │  via a 64-deep buffered channel (drops on overflow)
    ▼
per-client event loop (eventsHandler)
    │  reads from subscriber channel
    │  stream.Send(evt) → WriteEvent → ResponseWriter + Flush
    ▼
browser (DataStar SDK)
    parses SSE event → patches DOM (prepend feed item)
    or patches signals (update subscriber count, total events)
```

**Feature mapping:**

| Example behavior          | go-sse API                                                            |
| ------------------------- | --------------------------------------------------------------------- |
| All tabs see same events  | `Broadcaster.Broadcast` / `BroadcastMany` — non-blocking fan-out      |
| "N clients connected"     | `OnSubscribe` / `OnUnsubscribe` callbacks + `SubscriberCount()`       |
| "Alerts only" filter      | `SubscribeFilter(func(evt) bool)` — predicate under fan-out read lock |
| Reconnect replay          | `EventStore` (memStore) + `Replay` via `Last-Event-ID` header         |
| Connection stays alive    | `Stream.Heartbeat(ctx, interval)` — sends SSE comment every 15s       |
| Graceful drain on SIGINT  | `Broadcaster.Shutdown(ctx)` — drains buffers, then closes channels    |
| Health/status snapshot    | `Broadcaster.Health()` — value-type for k8s liveness/readiness probes |
| DataStar keyed data lines | `KeyedLines` + `SendLines` / `SendKeyed`                              |

**Shutdown sequence:**

1. `SIGINT` / `SIGTERM` received → context cancelled → producer goroutine exits
2. `httpServer.Shutdown(ctx)` drains active HTTP requests (SSE connections close)
3. `broadcaster.Shutdown(ctx)` marks the hub as draining (rejects new `Subscribe` calls), waits for subscriber buffers to empty, then closes all channels
4. If the deadline (5s) fires before the drain completes, the broadcaster returns a `context.DeadlineExceeded` error wrapped with code `sse.shutdown_drain_deadline_exceeded`

The example source is split into focused files: `main.go` (server setup + embed), `store.go` (in-memory ring buffer), `producer.go` (event generation), `handlers.go` (HTTP handlers + broadcaster wiring). See [`VERIFY.md`](example/datastar/VERIFY.md) for a manual browser verification checklist.

## Companion Libraries

- [go-error-family](https://github.com/larsartmann/go-error-family) — structured error wrapping with severity categories (used by go-sse for error codes)
- [go-branded-id](https://github.com/larsartmann/go-branded-id) — phantom-type branded IDs (used by go-sse for `EventID`)
- [httputil](https://github.com/larsartmann/httputil) — HTTP middleware utilities (compression, client IP, capabilities) that pair naturally with SSE handlers

## License

MIT
