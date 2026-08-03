# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to TODO_LIST.md.
> Completed work lives in CHANGELOG.md, not here.

## Sequencing

| Horizon | Theme                                                                   | Trigger to advance                                             |
| ------- | ----------------------------------------------------------------------- | -------------------------------------------------------------- |
| Now     | [Production readiness](#1-production-readiness)                         | First external consumer asks for shutdown or observability     |
| Next    | [Developer experience](#2-developer-experience)                         | Concrete client consumer or `EventStore`-shape request appears |
| Later   | [Spec compliance & extensibility](#3-spec-compliance-and-extensibility) | Spec amendment or concrete extension need                      |
| Parked  | [Parked decisions](#4-parked-decisions)                                 | Trigger criteria in the linked brainstorming docs fire         |

## 1. Production readiness

go-sse ships the four core pieces (wire format, connection, fan-out, replay).
Before calling it production-grade, explore what production consumers need.

**Exit criteria:** graceful-shutdown helper, configurable subscriber buffer,
and a scale profile exist; subscribers are observable beyond the existing
OnSubscribe/OnUnsubscribe hooks. The remaining explorations here are design
questions with multiple viable answers, not yet bounded enough to be tasks:

- Backpressure policy options beyond drop-on-full (block vs spill)
- Observability: metrics/structured logging beyond the OnSubscribe/OnUnsubscribe hooks

Bounded work extracted from this theme has moved to TODO_LIST.md:
graceful-shutdown helper, configurable subscriber buffer size, scale profiling.

## 2. Developer experience

Lower the barrier to adoption:

- Client-side `Dial` helper (the library is server-only today; deferred until a concrete client consumer exists, on the same don't-pre-build-for-imagined-consumers principle as the module-boundary decisions)
- Batteries-included `EventStore` implementations (in-memory, Redis)

## 3. Spec compliance and extensibility

Stay aligned with the SSE spec and explore extensions:

- SSE extension fields (CLTY, custom fields)
- Full HTTP/2 and HTTP/3 streaming verification
- Whether `LastEventID` should validate via `ParseEventID`

## 4. Parked decisions

Decisions examined and deferred, with explicit re-open triggers. Full
analysis lives in `docs/brainstorming/`; this section is an index, not a
decision log.

- **Split into `common` (wire format) + `server` (Stream, Broadcaster, Replay)?** Deferred 2026-07-25. The wire/server seam is exercised in production — 2 of 4 real consumers import go-sse wire-only and roll their own HTTP scaffolding. But the flat layout costs them ~zero today: `net/http` is stdlib, and `brandid`/`errorfamily` are genuinely used by the wire path. Re-open when **any** of these fires: a `client/` package is being written; the server gains a non-stdlib dependency wire-only consumers shouldn't transitively pull in (e.g. a Redis event store); a wire-only consumer asks to pin `common`; or a third wire-only consumer appears (2 → 3 turns a coincidence into an archetype). Full analysis: [docs/brainstorming/2026-07-25_client-server-common-submodule-split.md](docs/brainstorming/2026-07-25_client-server-common-submodule-split.md).
- **Export the unexported `fanOut[T]` hub?** Resolved (v0.2.0): no. No consumer needs it yet, and the generic `Broadcaster[T]` already serves "fan-out any type" for SSE consumers. Exporting would commit to API stability prematurely. Revisit when a concrete non-SSE use case emerges.

## 5. Raw ideas

Unexamined ideas — too early for a theme, not analyzed enough to be a parked
decision. Promoted to a numbered theme when bounded; dropped if ruled out.

- Topic/channel-based multi-broadcaster routing (multiple named fan-out hubs behind one entry point). No consumer has asked for this yet. The predicate-based filtering approach (`SubscribeFilter` + `ReplayFiltered`) solves the real consumer need (DiscordSync's per-channel/per-guild filtering) without the complexity of named hubs or wildcard matching. Revisit if a consumer needs true multi-hub routing rather than predicate filtering on a single hub.
- Optional `di/` subpackage providing samber/do v2 lifecycle adapters (`Shutdowner`, `Healthchecker`) for `Broadcaster` and `Stream`. See [docs/brainstorming/2026-08-03_samber-do-lifecycle-integration.md](docs/brainstorming/2026-08-03_samber-do-lifecycle-integration.md).

## Non-goals

Things we are deliberately NOT pursuing and why:

- **WebSockets:** SSE is a fundamentally different transport (one-way, HTTP-based). Adding WebSocket support would expand scope beyond the library's purpose.
- **CQRS dispatch hooks / event bus integration:** Consumers build domain layers on top. Opinions about dispatch belong in the consumer, not the transport.
- **Dashboard server / routes / HTML templates:** This is a library, not a framework or application.
- **Payload-format opinions:** Strings, JSON, HTML fragments are all valid `data`. The library serializes whatever you give it.
- **`Broadcaster.ServeSSE` handler:** A convenience handler would bake in opinions about heartbeat interval, replay behavior, and event-loop structure. These belong in the consumer's handler, not the library. The `example/` package shows the canonical pattern using `Stream` + `Broadcaster.Subscribe`.
