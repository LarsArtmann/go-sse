# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to TODO_LIST.md.

## Themes

### 1. Production readiness

go-sse ships the four core pieces (wire format, connection, fan-out, replay).
Before calling it production-grade, explore what production consumers need:

- Graceful shutdown helper: drain subscribers on SIGTERM before closing broadcasters
- Configurable subscriber buffer size (currently hardcoded at 64 in `fanout.go`)
- Backpressure policy options beyond drop-on-full (block vs spill)
- Memory characteristics at scale: profile 64-buffer x N subscribers
- Observability: metrics/structured logging beyond the OnSubscribe/OnUnsubscribe hooks

### 2. Developer experience

Lower the barrier to adoption:

- Client-side `Dial` helper (the library is server-only today; deferred until a concrete client consumer exists, on the same don't-pre-build-for-imagined-consumers principle as the module-boundary decisions)
- Batteries-included `EventStore` implementations (in-memory, Redis)

> Realized in 0.2.0: `example/` runnable server, `Event.String()` debug helper,
> `SendJSON` convenience method, Go `Example` functions in godoc.
>
> Realized in 0.3.0: `KeyedLines` + `SendLines` + `WriteKeyedLines` +
> `SendKeyed` + `JSONSignals` — general-purpose keyed-data-line helpers that
> make the library a first-class SSE backend for [DataStar](https://data-star.dev)
> and similar keyed-data-line protocols, without coupling the library to any
> specific framework.

### 3. Spec compliance and extensibility

Stay aligned with the SSE spec and explore extensions:

- SSE extension fields (CLTY, custom fields)
- Full HTTP/2 and HTTP/3 streaming verification
- Whether `LastEventID` should validate via `ParseEventID`

> Realized in 0.2.0: batch/multi-event send via `Broadcaster.BroadcastMany`.

### 4. Module boundaries and reusability

The wire-format code (`event.go`) is already `net/http`-free; only `stream.go`
touches HTTP, so the conceptual seams exist even though the package is flat.
Two boundary questions have been examined:

- **Export the unexported `fanOut[T]` hub?** Resolved (v0.2.0): no. No consumer needs it yet, the generic `Broadcaster[T]` already serves "fan-out any type" for SSE consumers, and exporting would commit to API stability prematurely. Revisit when a concrete non-SSE use case emerges.
- **Split into `common` (wire format) + `server` (Stream, Broadcaster, Replay)?** Analyzed 2026-07-25, deferred. A consumer audit found that **2 of 4 real consumers import go-sse wire-only** — they roll their own HTTP scaffolding around `WriteEvent`/`ContentType` and never touch `Stream`/`Broadcaster`/`Replay`. So the wire/server seam is exercised in production, not hypothetical. But `net/http` is stdlib and `brandid`/`errorfamily` are genuinely used by the wire path, so the flat layout costs wire-only consumers ~zero today, and all consumers are internal. The cheap hedge — keeping `event.go` functions strictly `io.Writer`-based — is already a contract being exercised in the wild. Re-open when a trigger fires: a `client/` is being written; the server gains a non-stdlib dependency wire-only consumers shouldn't transitively pull in (e.g. a Redis event store); a wire-only consumer asks to pin `common`; or a third wire-only consumer appears. Full analysis: [docs/brainstorming/2026-07-25_client-server-common-submodule-split.md](docs/brainstorming/2026-07-25_client-server-common-submodule-split.md).

Still-open raw idea:

- Topic/channel-based multi-broadcaster routing (multiple named fan-out hubs behind one entry point). No consumer has asked for this yet.

## Non-goals

Things we are deliberately NOT pursuing and why:

- **WebSockets:** SSE is a fundamentally different transport (one-way, HTTP-based). Adding WebSocket support would expand scope beyond the library's purpose.
- **CQRS dispatch hooks / event bus integration:** Consumers build domain layers on top. Opinions about dispatch belong in the consumer, not the transport.
- **Dashboard server / routes / HTML templates:** This is a library, not a framework or application.
- **Payload-format opinions:** Strings, JSON, HTML fragments are all valid `data`. The library serializes whatever you give it.
- **`Broadcaster.ServeSSE` handler:** A convenience handler would bake in opinions about heartbeat interval, replay behavior, and event-loop structure. These belong in the consumer's handler, not the library. The `example/` package shows the canonical pattern using `Stream` + `Broadcaster.Subscribe`.
