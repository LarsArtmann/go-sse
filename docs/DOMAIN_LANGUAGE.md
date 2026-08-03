# Domain Language

The ubiquitous vocabulary of `go-sse`. These terms come from the
[SSE specification](https://html.spec.whatwg.org/multipage/server-sent-events.html)
and this library's transport abstractions. Using them consistently keeps code and
docs aligned.

## Glossary

| Term               | Definition                                                                                                                                                                     | Where used                                 |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------ |
| Event              | A single Server-Sent Event: an event name, data payload, optional id, and optional retry hint. Maps to one wire-format block.                                                  | `Event` in `event.go`                      |
| EventID            | A branded identifier for the `id:` field. Arbitrary server-defined string (NOT a ULID). Sent back by the browser on reconnect.                                                 | `EventID` in `event.go`                    |
| Wire format        | The textual SSE framing: `event:`, `data:`, `id:`, `retry:` lines terminated by a blank line.                                                                                  | `WriteEvent` in `event.go`                 |
| data field         | The payload. Multi-line data is split so each line gets its own `data:` prefix (required by the spec).                                                                         | `splitLines` in `event.go`                 |
| Comment frame      | A line beginning with `:`. Browsers ignore it; used to send heartbeats that reset proxy idle timers without delivering an event.                                               | `WriteHeartbeat` in `event.go`             |
| Heartbeat          | Periodic comment-frame ping sent to keep a connection alive through Nginx/Cloudflare/AWS ALB idle timeouts (typically 30-60s).                                                 | `Stream.Heartbeat` in `stream.go`          |
| Retry              | The `retry:` field. Tells the browser how many milliseconds to wait before reconnecting after a drop. Persists until overwritten.                                              | `WriteRetry` in `event.go`, `Event.Retry`  |
| Last-Event-ID      | HTTP request header the browser sends on reconnect, containing the last `id:` it received. Drives replay.                                                                      | `Stream.LastEventID` in `stream.go`        |
| Stream             | A single SSE connection's lifecycle: headers, status, mutex-guarded writes, heartbeat, disconnect hooks, and context cancellation.                                             | `Stream` in `stream.go`                    |
| Broadcaster        | Generic subscriber fan-out: one `Broadcast` reaches all subscribed channels. Public wrapper over `fanOut`.                                                                     | `Broadcaster[T]` in `broadcaster.go`       |
| fanOut             | The unexported, transport-agnostic subscriber hub. Owns the subscriber map, channel pointer identity, and non-blocking send.                                                   | `fanOut[T]` in `fanout.go`                 |
| Subscriber         | A buffered channel registered with the `fanOut` that receives broadcast messages. Default depth 64, configurable via `WithBufferSize`.                                         | `fanOut[T].Subscribe` in `fanout.go`       |
| Drop policy        | Non-blocking broadcast: when a subscriber's buffer is full, the message is silently dropped for that subscriber. By design.                                                    | `fanOut[T].Broadcast` in `fanout.go`       |
| Predicate          | A `func(T) bool` checked before the non-blocking send during fan-out. When non-nil on a subscriber, only matching messages enter the buffer. Must be pure, fast, non-blocking. | `subscriber[T].pred` in `fanout.go`        |
| SubscribeFilter    | Predicate-based subscription: creates a subscriber whose channel receives only messages matching the predicate. `Subscribe()` is `SubscribeFilter(nil)`.                       | `fanOut[T].SubscribeFilter` in `fanout.go` |
| Replay             | Sending missed events (those after a given `Last-Event-ID`) to a reconnecting client before resuming live delivery.                                                            | `Replay` in `replay.go`                    |
| EventStore         | Consumer-implemented interface that returns events strictly after a given id, ascending. Backs `Replay`.                                                                       | `EventStore` in `replay.go`                |
| FilteredEventStore | An `EventStore` that additionally supports pushing a predicate into its retrieval query, so the replay budget is spent on matching events only.                                | `FilteredEventStore` in `replay.go`        |
| ReplayFiltered     | Replays only events matching a predicate. Uses `FilteredEventStore` (efficient path) when available; falls back to `EventsAfter` + post-filter (correct path).                 | `ReplayFiltered` in `replay.go`            |
| Branded ID         | A phantom-type-wrapped string (`brandid.ID[eventBrand, string]`) that prevents accidental cross-assignment with other string IDs.                                              | `EventID` in `event.go`                    |

## Bounded context

`go-sse` has one bounded context: **SSE transport**. The terms above do not
carry domain meaning — they are transport/spec vocabulary. Consumers layer their
own domain language on top (e.g. a "todoCreated" event name is the consumer's
domain term, not this library's).
