# Domain Language

The ubiquitous vocabulary of `go-sse`. These terms come from the
[SSE specification](https://html.spec.whatwg.org/multipage/server-sent-events.html)
and this library's transport abstractions. Using them consistently keeps code and
docs aligned.

## Glossary

| Term          | Definition                                                                                                                         | Where used                                   |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------- |
| Event         | A single Server-Sent Event: an event name, data payload, optional id, and optional retry hint. Maps to one wire-format block.      | `Event` (`event.go:74`)                      |
| EventID       | A branded identifier for the `id:` field. Arbitrary server-defined string (NOT a ULID). Sent back by the browser on reconnect.     | `EventID` (`event.go:26`)                    |
| Wire format   | The textual SSE framing: `event:`, `data:`, `id:`, `retry:` lines terminated by a blank line.                                      | `WriteEvent` (`event.go:96`)                 |
| data field    | The payload. Multi-line data is split so each line gets its own `data:` prefix (required by the spec).                             | `splitLines` (`event.go:159`)                |
| Comment frame | A line beginning with `:`. Browsers ignore it; used to send heartbeats that reset proxy idle timers without delivering an event.   | `WriteHeartbeat` (`event.go:140`)            |
| Heartbeat     | Periodic comment-frame ping sent to keep a connection alive through Nginx/Cloudflare/AWS ALB idle timeouts (typically 30-60s).     | `Stream.Heartbeat` (`stream.go:152`)         |
| Retry         | The `retry:` field. Tells the browser how many milliseconds to wait before reconnecting after a drop. Persists until overwritten.  | `WriteRetry` (`event.go:149`), `Event.Retry` |
| Last-Event-ID | HTTP request header the browser sends on reconnect, containing the last `id:` it received. Drives replay.                          | `Stream.LastEventID` (`stream.go:137`)       |
| Stream        | A single SSE connection's lifecycle: headers, status, mutex-guarded writes, heartbeat, disconnect hooks, and context cancellation. | `Stream` (`stream.go:35`)                    |
| Broadcaster   | Generic subscriber fan-out: one `Broadcast` reaches all subscribed channels. Public wrapper over `fanOut`.                         | `Broadcaster[T]` (`broadcaster.go:23`)       |
| fanOut        | The unexported, transport-agnostic subscriber hub. Owns the subscriber map, channel pointer identity, and non-blocking send.       | `fanOut[T]` (`fanout.go:17`)                 |
| Subscriber    | A buffered channel (depth 64) registered with the `fanOut` that receives broadcast messages.                                       | `fanOut[T].Subscribe` (`fanout.go:39`)       |
| Drop policy   | Non-blocking broadcast: when a subscriber's buffer is full, the message is silently dropped for that subscriber. By design.        | `fanOut[T].Broadcast` (`fanout.go:89`)       |
| Predicate     | A `func(T) bool` checked before the non-blocking send during fan-out. When non-nil on a subscriber, only matching messages enter the buffer. Must be pure, fast, non-blocking. | `subscriber[T].pred` (`fanout.go:72`) |
| SubscribeFilter | Predicate-based subscription: creates a subscriber whose channel receives only messages matching the predicate. `Subscribe()` is `SubscribeFilter(nil)`. | `fanOut[T].SubscribeFilter` (`fanout.go:142`) |
| Replay        | Sending missed events (those after a given `Last-Event-ID`) to a reconnecting client before resuming live delivery.                | `Replay` (`replay.go:21`)                    |
| EventStore    | Consumer-implemented interface that returns events strictly after a given id, ascending. Backs `Replay`.                           | `EventStore` (`replay.go:9`)                 |
| FilteredEventStore | An `EventStore` that additionally supports pushing a predicate into its retrieval query, so the replay budget is spent on matching events only. | `FilteredEventStore` (`replay.go:25`) |
| ReplayFiltered | Replays only events matching a predicate. Uses `FilteredEventStore` (efficient path) when available; falls back to `EventsAfter` + post-filter (correct path). | `ReplayFiltered` (`replay.go:70`) |
| Branded ID    | A phantom-type-wrapped string (`brandid.ID[eventBrand, string]`) that prevents accidental cross-assignment with other string IDs.  | `EventID` (`event.go:26`)                    |

## Bounded context

`go-sse` has one bounded context: **SSE transport**. The terms above do not
carry domain meaning — they are transport/spec vocabulary. Consumers layer their
own domain language on top (e.g. a "todoCreated" event name is the consumer's
domain term, not this library's).
