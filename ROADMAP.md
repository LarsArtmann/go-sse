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

- Client-side `Dial` helper (currently server-only)
- Batteries-included `EventStore` implementations (in-memory, Redis)

> Realized in 0.1.x: `example/` runnable server, `Event.String()` debug helper,
> `SendJSON` convenience method, Go `Example` functions in godoc.

### 3. Spec compliance and extensibility

Stay aligned with the SSE spec and explore extensions:

- SSE extension fields (CLTY, custom fields)
- Full HTTP/2 and HTTP/3 streaming verification
- Whether `LastEventID` should validate via `ParseEventID`

> Realized in 0.1.x: batch/multi-event send via `Broadcaster.BroadcastMany`.

### 4. Reusability of the fan-out hub

The unexported `fanOut[T]` is transport-agnostic. Explore:

- Exporting it for non-SSE fan-out use cases
- Topic/channel-based multi-broadcaster routing
- Whether the split into Broadcaster (SSE) + fanOut (generic) should become a public type boundary

> **Decision (v0.1.x):** Keep `fanOut[T]` unexported. No consumer needs it
> yet, and exporting commits to API stability prematurely. Revisit when a
> concrete non-SSE use case emerges. The generic `Broadcaster[T]` already
> serves the "fan-out any type" need for SSE consumers.

## Non-goals

Things we are deliberately NOT pursuing and why:

- **WebSockets:** SSE is a fundamentally different transport (one-way, HTTP-based). Adding WebSocket support would expand scope beyond the library's purpose.
- **CQRS dispatch hooks / event bus integration:** Consumers build domain layers on top. Opinions about dispatch belong in the consumer, not the transport.
- **Dashboard server / routes / HTML templates:** This is a library, not a framework or application.
- **Payload-format opinions:** Strings, JSON, HTML fragments are all valid `data`. The library serializes whatever you give it.
