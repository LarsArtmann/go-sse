# Reconnection and Retry in go-sse

How reconnection actually works in a Server-Sent Events system built on this
library, layer by layer. There is no `retry` loop anywhere in go-sse — the
word "retry" in the API (`Event.Retry`, `WriteRetry`) means the _browser's_
reconnection hint, not a client-side retry of a failed operation. Reliability
comes from five cooperating layers, each owned by a different component.

## The five layers

| # | Layer                      | Owner           | What it recovers from                                    |
| - | -------------------------- | --------------- | -------------------------------------------------------- |
| 1 | Browser `EventSource`      | Client          | Any dropped connection — automatically, forever          |
| 2 | `retry:` hint              | Server → client | Paces reconnection attempts after a failure              |
| 3 | `Last-Event-ID` + `Replay` | Server          | Events missed while the client was disconnected          |
| 4 | `Heartbeat`                | Server          | Idle connections killed by proxies/load balancers        |
| 5 | `Shutdown` drain           | Your process    | Clean handover on deploys — no events lost mid-broadcast |

### 1. The browser reconnects (you get this for free)

`EventSource` is a self-healing client: when the connection drops, the browser
reconnects on its own. go-sse's job ends at writing correct frames — a failed
`Stream.Send` means the connection is dead, and the correct response is to drop
it, not to retry. Retrying a partial write would re-emit bytes already on the
wire and corrupt the SSE frame stream. This is why connection-death errors
(`sse.send_failed`, `sse.write_failed`) are classified `Transient` in
`go-error-family` terms: "transient" here means _the client will recover by
reconnecting_, not "try the same socket again".

### 2. The `retry:` hint (server sets the pace)

After a failure, the browser waits before reconnecting (its default is a few
seconds with exponential backoff on repeated failures). The server can set or
adjust this interval:

```go
evt := sse.Event{Event: "feed", Data: "...", Retry: 3000} // milliseconds
```

or send a bare reconnection-time frame with `sse.WriteRetry(w, 3000)`. The
value is sticky connection state: the browser keeps applying it until a later
`retry:` field changes it.

### 3. `Last-Event-ID` + `Replay` (no missed events)

This is the layer that makes reconnection _lossless_. Every event can carry an
`id:`; the browser remembers the last one it saw and sends it back in the
`Last-Event-ID` header on reconnect. The server reads it and replays the gap
from a store:

```go
func eventsHandler(w http.ResponseWriter, r *http.Request) {
    stream := sse.NewStream(w, r)
    lastID := sse.LastEventIDFromRequest(r)
    if _, err := sse.Replay(stream, store, lastID); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    // ... then subscribe and forward live events
}
```

`EventStore` is the interface you implement (ring buffer, database, whatever);
`Replay` writes everything after `lastID` before the live subscription starts.
An empty `LastEventID` means "initial connection" — replay everything your
store's semantics dictate. IDs are validated with `ParseEventID` (the spec
forbids NUL, LF, CR in the header value); `MustParseEventID` is for constants
and tests only.

See `example/datastar/` for the full pattern (store + replay + live fan-out).

### 4. `Heartbeat` (the connection survives idle periods)

Proxies and load balancers kill idle connections long before the browser
notices. A comment frame keeps them open:

```go
go stream.Heartbeat(ctx, 15*time.Second)
```

`Heartbeat` writes `": heartbeat"` on the interval; browsers ignore comment
lines, but intermediary idle timers reset. Without it, "no events for five
minutes" and "connection died five minutes ago" look identical to the network
— and the client only finds out on its next (re)connect.

### 5. `Shutdown` drain (clean deploys)

On SIGTERM, don't just close the listener: drain the broadcaster so in-flight
events reach their subscribers before the process exits.

```go
ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
defer cancel()
if err := broadcaster.Shutdown(ctx); err != nil {
    broadcaster.Close() // deadline exceeded — hard shutdown
}
```

`Shutdown` stops accepting new subscribers, waits for every subscriber's
buffer to empty, then closes the channels. It preserves
`errors.Is(err, context.DeadlineExceeded)` for callers that retry with a
fresh context.

## What go-sse deliberately does not provide

- **No client-side retry loop.** A failed `Stream.Send` is a dead connection;
  layer 1 (the browser) owns recovery. Wrapping sends in a retry wrapper
  corrupts the frame stream.
- **No EventStore implementation.** The store's retention and retry policy is
  domain logic; go-sse defines the interface and the replay semantics.
- **No backoff policy.** The browser owns reconnection pacing; the `retry:`
  hint is the only server-side influence.

Full rationale: [docs/brainstorming/2026-08-07_go-retry-adoption-evaluation.md](../brainstorming/2026-08-07_go-retry-adoption-evaluation.md).
