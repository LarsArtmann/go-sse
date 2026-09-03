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
| —       | [Raw ideas](#5-raw-ideas)                                               | Analyzed enough to become a theme or a parked decision         |

## 1. Production readiness

go-sse ships the four core pieces (wire format, connection, fan-out, replay).
Before calling it production-grade, explore what production consumers need.

**Exit criteria:** graceful-shutdown helper, configurable subscriber buffer,
and a scale profile exist; subscribers are observable beyond the existing
OnSubscribe/OnUnsubscribe hooks.

All four criteria are met: graceful shutdown (`Broadcaster.Shutdown(ctx)`),
configurable buffer (`WithBufferSize[T]`), structured observability
(`Broadcaster.Health()` returning `BroadcasterHealth`), and a scale profile
([docs/performance/scale-profile.md](docs/performance/scale-profile.md) —
conclusion: the default 64-buffer and non-blocking drop policy are
well-calibrated; no change needed). The remaining explorations here are design
questions with multiple viable answers, not yet bounded enough to be tasks:

- Backpressure policy options beyond drop-on-full (block vs spill)
- Observability: metrics/structured logging beyond `Health()` and the
  OnSubscribe/OnUnsubscribe hooks (e.g., drop counters, per-subscriber stats)
- Whether predicate panic recovery should expose an `OnPredicatePanic` callback
  or log hook — currently silent (see v0.4.0 release report Q2)

## 2. Developer experience

Lower the barrier to adoption:

- Client-side `Dial` helper (the library is server-only today; deferred until a concrete client consumer exists, on the same don't-pre-build-for-imagined-consumers principle as the module-boundary decisions)
- Batteries-included `EventStore` implementations (in-memory, Redis)

## 3. Spec compliance and extensibility

Stay aligned with the SSE spec and explore extensions:

- SSE extension fields (CLTY, custom fields)
- Full HTTP/2 and HTTP/3 streaming verification

(Note: the former "should `LastEventID` validate via `ParseEventID`" question is resolved — `LastEventIDFromRequest` has validated through `ParseEventID` since v0.1.0.)

## 4. Parked decisions

Decisions examined and deferred, with explicit re-open triggers. Full
analysis lives in `docs/brainstorming/`; this section is an index, not a
decision log.

- **Split into `common` (wire format) + `server` (Stream, Broadcaster, Replay)?** Deferred 2026-07-25. The wire/server seam is exercised in production — 2 of 4 real consumers import go-sse wire-only and roll their own HTTP scaffolding. But the flat layout costs them ~zero today: `net/http` is stdlib, and `brandid`/`errorfamily` are genuinely used by the wire path. Re-open when **any** of these fires: a `client/` package is being written; the server gains a non-stdlib dependency wire-only consumers shouldn't transitively pull in (e.g. a Redis event store); a wire-only consumer asks to pin `common`; or a third wire-only consumer appears (2 → 3 turns a coincidence into an archetype). Full analysis: [docs/brainstorming/2026-07-25_client-server-common-submodule-split.md](docs/brainstorming/2026-07-25_client-server-common-submodule-split.md).
- **Adopt [`go-retry`](https://github.com/LarsArtmann/go-retry) for backoff/retry?** Deferred 2026-08-07: no. The dependency cost is genuinely near zero (its only dependency is `go-error-family v0.10.0`, already required at that exact version — adopting adds no new modules to the build graph). The blocker is that **go-sse has no retryable operation**: a failed `Stream.Send` means a dead or partially-written connection (retrying corrupts frames and would sleep while holding the heartbeat mutex), `EventStore` retry policy belongs to the consumer's store, the `Shutdown` drain is a condition-wait where backoff only raises shutdown latency, and `Broadcast`'s drop-on-full is a documented invariant. Worse, go-sse classifies connection-death errors as `Transient`, which is exactly what `errorfamily.IsRetryable` retries — so the default config would retry a broken pipe. Also collides with the existing `Event.Retry`/`WriteRetry` wire field (a browser reconnect hint, unrelated concept). Re-open when **any** of these fires: a client `Dial`/reconnect helper is being written (§2 — the genuine use case); an in-tree `EventStore` with a network backend ships (go-sse would then own the failing I/O); or upstream reconciles `errorfamily.RetryPolicy` with `retry.Config` and fixes the three reproduced `ComputeDelay` panics. Full analysis: [docs/brainstorming/2026-08-07_go-retry-adoption-evaluation.md](docs/brainstorming/2026-08-07_go-retry-adoption-evaluation.md).
- **Export the unexported `fanOut[T]` hub?** Resolved (v0.2.0): no. No consumer needs it yet, and the generic `Broadcaster[T]` already serves "fan-out any type" for SSE consumers. Exporting would commit to API stability prematurely. Revisit when a concrete non-SSE use case emerges.

## 5. Raw ideas

Unexamined ideas — too early for a theme, not analyzed enough to be a parked
decision. Promoted to a numbered theme when bounded; dropped if ruled out.
(Deferred design notes rescued from archived session reports during the
2026-09-03 docs-health pass are collected at the end of this list.)

- Topic/channel-based multi-broadcaster routing (multiple named fan-out hubs behind one entry point). No consumer has asked for this yet. The predicate-based filtering approach (`SubscribeFilter` + `ReplayFiltered`) solves the real consumer need (DiscordSync's per-channel/per-guild filtering) without the complexity of named hubs or wildcard matching. Revisit if a consumer needs true multi-hub routing rather than predicate filtering on a single hub.
- Optional `di/` subpackage providing samber/do v2 lifecycle adapters (`Shutdowner`, `Healthchecker`) for `Broadcaster` and `Stream`. See [docs/brainstorming/2026-08-03_samber-do-lifecycle-integration.md](docs/brainstorming/2026-08-03_samber-do-lifecycle-integration.md). The brainstorming doc's Option C (core primitives + documented pattern) was adopted: `Shutdown(ctx)` and `Health()` ship in core, the samber/do adapter is left to consumer composition roots. Revisit if a concrete consumer asks for a go-sse-provided adapter (Option B trigger).
- Ecosystem-wide dependency policy for the `larsartmann/*` modules (one shared doc or per-repo AGENTS.md sections): when to pin vs track, how to treat brand-new majors, and a shared rule against tests coupled to one upstream version's incidental behavior. Motivated by the 2026-07-27 go-branded-id regression: brand-aware `String()` shipped v0.3.0–v0.3.2, was dropped in v0.3.3/v0.4.0 (breaking a go-sse test), and returned in v0.5.0. Source: [docs/status/archived/2026-07-27_10-26_removed-brand-name-test-and-self-review.md](docs/status/archived/2026-07-27_10-26_removed-brand-name-test-and-self-review.md) item 44.
- Replay pagination for huge gaps: `EventsAfter` with a limit/offset (or cursor) so a client reconnecting after a long outage cannot force an unbounded replay. API design sketch first, in-scope decision second. Source: 2026-08-29 16-36 report §f24.
- `WithDrainPollInterval` option and a richer `ShutdownResult` (per-subscriber drain stats) — deferred twice in the v0.4.0-era sessions; revisit only if a consumer reports slow `Shutdown` drain visibility problems. Source: 2026-08-03 20-20 report.
- Example-servers as flake apps (`nix run .#datastar` / `.#htmx` / `.#server`) — repeatedly requested, repeatedly deferred; decide once. Source: 2026-08-29 13-53 report §f37.
- CSP headers (and an SRI posture) for the examples' vendored static assets — vendoring removed the CDN/SRI concern, but no Content-Security-Policy is served. Source: 2026-08-29 13-53 report §f38.
- `docs/status/INDEX.md` (date → one-sentence outcome → disposition per report) — only worth it if generated from git history; a hand-maintained index would drift by design. Source: 2026-08-29 13-53 report §f29.
- `docs/guides/getting-started.md` distinct from the README quickstart (README is a sales page per the doc-file contract; a step-by-step guide is a different artifact). Source: 2026-08-29 16-36 report §f50.

## Non-goals

Things we are deliberately NOT pursuing and why:

- **WebSockets:** SSE is a fundamentally different transport (one-way, HTTP-based). Adding WebSocket support would expand scope beyond the library's purpose.
- **CQRS dispatch hooks / event bus integration:** Consumers build domain layers on top. Opinions about dispatch belong in the consumer, not the transport.
- **Dashboard server / routes / HTML templates:** This is a library, not a framework or application.
- **Payload-format opinions:** Strings, JSON, HTML fragments are all valid `data`. The library serializes whatever you give it.
- **`Broadcaster.ServeSSE` handler:** A convenience handler would bake in opinions about heartbeat interval, replay behavior, and event-loop structure. These belong in the consumer's handler, not the library. The `example/` package shows the canonical pattern using `Stream` + `Broadcaster.Subscribe`.
