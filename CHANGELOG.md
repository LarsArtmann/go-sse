# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- `ExampleReplayFiltered` — godoc example demonstrating predicate-based reconnection replay. Completes example coverage for every public replay/filter API.
- `TestSubscribeFilter_DropPolicyRespected` — verifies a filtered subscriber with a full buffer drops matching events (non-matching events never enter the buffer), confirming the filter+drop interaction.
- `TestSubscribeFilter_BroadcastManyMixedSubscribers` — verifies correct event partition when half the subscribers have predicates and half do not across a single `BroadcastMany` batch.
- `BenchmarkMemoryPerSubscriber` — measures steady-state heap memory per subscriber at 100/1k/10k subscribers (~4 KiB each, buffer-dominated).
- `docs/performance/scale-profile.md` — memory and latency characterization at 100/1k/10k subscribers. Conclusion: the default 64-buffer and non-blocking drop policy are well-calibrated; no change needed.

### Fixed

- `example/datastar/main.go` progress bar corrected: `data-bind:style` → `data-style:width="$progress + '%'"`. `data-bind` is DataStar's form-element two-way binding; CSS styles use `data-style`. Verified against the DataStar v1.0.2 attributes reference.
- `example/datastar/main.go` import changed from `encoding/json/v2` to `encoding/json`. The example is a teaching tool meant to be copied by consumers who may not set `GOEXPERIMENT=jsonv2`; stable `encoding/json` keeps the example portable.
- `broadcaster.go` `NewBroadcaster` removed an unnecessary explicit type argument (`newFanOut[T]` → `newFanOut`) flagged by gopls `infertypeargs`.
- `.envrc` now exports `GOEXPERIMENT=jsonv2` alongside `GOWORK=off`, so `buildflow`, `gopls`, and direct `go` invocations launched outside the Nix devShell inherit the flag via direnv (AGENTS.md already documented this as a gotcha).
- README.md and doc.go predicate panic-policy documentation corrected: both now state that panicking predicates are recovered and treated as non-matches (matching v0.4.0 code behavior). The v0.4.0 tag permanently contains the old "crashes the broadcaster" wording; this fix lands on master for consumers reading HEAD.

## [0.4.0] - 2026-08-03

### Added

- `Broadcaster.SubscribeFilter(pred func(T) bool) <-chan T` — predicate-based subscription that delivers only events matching the predicate to the subscriber's channel. The predicate is checked before the non-blocking send, so irrelevant events never enter the buffer. `Subscribe()` is now `SubscribeFilter(nil)`. Zero overhead for unfiltered subscribers (nil check only).
- `FilteredEventStore` interface — implemented by event stores that can push a predicate into their retrieval query, so the replay budget is spent entirely on matching events instead of being wasted on non-matching ones that get filtered post-hoc.
- `ReplayFiltered(stream *Stream, store EventStore, lastID EventID, pred func(Event) bool) (int, error)` — replays only events matching `pred`. If the store implements `FilteredEventStore`, the predicate is pushed into the store query (efficient). Otherwise falls back to `EventsAfter` + in-memory post-filter (correct). Nil pred delegates to `Replay`.
- `KeyedLines(key, value string) string` — prefixes every line of a multi-line value with `key `, producing the newline-joined string for `Event.Data`. Building block for keyed-data-line SSE protocols (DataStar, htmx extensions, etc.).
- `Stream.SendLines(eventName string, lines ...string) error` — convenience method that joins variadic arguments with `\n` into `Event.Data`, then delegates to `Send`. Composes with `KeyedLines` for multi-key events.
- `WriteKeyedLines(w io.Writer, eventType, key, value string) error` — wire-only helper (no `net/http` dependency) for consumers that use `WriteEvent` directly. Single-key convenience counterpart to `KeyedLines`.
- `Stream.SendKeyed(eventName, key, value string) error` — stream convenience for the most common single-key DataStar pattern (e.g., `patch-signals` with one `signals` key).
- `FuzzKeyedLines` fuzz test — panic-safety with arbitrary key/value inputs.
- `BenchmarkKeyedLines` — single-line and 100-line variants measuring allocation behavior.
- `BenchmarkSubscribeFilter_PredicateOverhead` — measures per-subscriber predicate call overhead at 1/100/1000 subscribers (unfiltered vs filtered).
- `TestSubscribeFilter_BroadcastManyRespectsPredicates` — verifies BroadcastMany honors subscriber predicates.
- `TestSubscribeFilter_PredicatePanicRecovered` — verifies a panicking predicate is recovered (treated as non-match) and does not crash the broadcaster.
- `TestSubscribeFilter_ShutdownDrainsFilteredSubscribers` — verifies Shutdown drain works correctly when subscribers have predicates.
- `TestIntegration_SubscribeFilter` — HTTP round-trip verifying non-matching events never reach the client.
- `TestIntegration_ReplayFiltered` — HTTP round-trip for ReplayFiltered covering both the FilteredEventStore (efficient) and plain EventStore (fallback) paths.
- `TestReplayFiltered_FallbackPredicatePanicRecovered` — verifies a panicking predicate in the fallback path is recovered and does not crash.
- DataStar wire-format integration test — HTTP round-trip asserting exact wire bytes.
- `Broadcaster.Shutdown(ctx context.Context) error` — graceful shutdown that stops accepting new subscribers, waits for every active subscriber's buffer to drain (consumers catch up), then closes all channels. Returns a wrapped context error (`sse.shutdown_drain_deadline_exceeded`) if the deadline fires before the drain completes; the caller can retry with a fresh context or fall back to `Close`. Preserves `errors.Is(err, context.Canceled)` / `context.DeadlineExceeded` for existing context-aware code.
- `Broadcaster.Health() BroadcasterHealth` — value-type snapshot of `Closed`, `Draining`, `SubscriberCount`, and `BufferSize`. Cheap (read-lock + struct copy) and safe from any goroutine, including health-check loops.
- `Option[T any]` and `WithBufferSize[T any](size int) Option[T]` — functional options for `NewBroadcaster`. Pass `WithBufferSize[T](256)` (or any positive integer) to override the per-subscriber channel capacity from the default 64. Non-positive values are silently ignored.
- `NewBroadcaster[T any](opts ...Option[T])` — variadic constructor accepting `Option` values. Existing zero-arg call sites are unchanged.
- `BroadcasterHealth` struct — exported for consumers wiring structured health checks (k8s liveness/readiness, load balancer probes).
- 13 new tests in `lifecycle_test.go` covering Shutdown (empty, drains, context cancel, rejects new subs during drain, idempotent, after close, concurrent unsubscribe), Health (initial, during operation, after close, buffer size), and `WithBufferSize` (applies, non-positive ignored).

### Changed

- Predicate calls (`SubscribeFilter` and `ReplayFiltered` fallback) now recover from panics. A panicking predicate is treated as a non-match (the event is skipped for that subscriber). This ensures one broken predicate cannot crash the broadcaster or replay loop.
- `replay_test.go` and `replay_filter_test.go` now use the existing `newTestStream` helper from `testhelpers_test.go` instead of duplicating the stream-setup boilerplate. The shared `errorResponseWriter` test fake moved from `replay_test.go` to `testhelpers_test.go` so any test file can use it; `newTestFailingStream(t)` was added to the helpers for write-failure paths.
- `filter_test.go` adopts the Go 1.26 `sync.WaitGroup.Go` helper in `TestSubscribeFilter_ConcurrentRace` instead of the manual `wg.Add` / `defer wg.Done` pair.
- `fanOut` now tracks a `draining` flag alongside the existing `subscribers = nil` closed sentinel, so `Shutdown` can reject new subscribers during a drain without conflating "shutting down" with "closed".
- `TestSubscribeFilter_ConcurrentRace` strengthened: asserts `received >= 500` (meaningful threshold) instead of just `> 0`.

## [0.3.0] - 2026-07-27

No user-facing code changes. Tagged as a checkpoint between the v0.2.1 patch
and the subsequent DataStar integration work.

## [0.2.1] - 2026-07-26

### Added

- CI `govulncheck` job and `fuzz` job (`FuzzWriteEvent`, `FuzzParseEventID` at 1m each).
- Integration tests: heartbeat comment-frame delivery over a real HTTP round-trip; `Last-Event-ID` reconnection replay over a real HTTP round-trip.
- Unit tests covering `eventBrand.Name()`, the `MustParseEventID` success path, and the `Stream.Heartbeat` write-error exit path.

### Changed

- `go-error-family` dependency bumped from `v0.8.0` to `v0.9.0`.
- Test coverage raised to 100% of statements (removed an unreachable dead branch in `splitLines`).
- Test modernization: `context.WithCancel(context.Background())` → `context.WithCancel(t.Context())`; `wg.Add` + `go func()` + `defer wg.Done()` → `wg.Go` in race tests.
- Doc/source/README examples now use `defer func() { _ = stream.Close() }()` (`Stream` satisfies `io.Closer`) instead of `defer stream.Close()`.

### Fixed

- Flaky tests: `TestIntegration_BroadcasterFanOut` and `TestStream_Heartbeat` no longer use `time.Sleep` — both wait deterministically on channel signals.

## [0.2.0] - 2026-07-24

### Changed

- **Breaking:** `Stream.SendHTML` renamed to `Stream.SendData` — the method sends any raw string, not just HTML; the old name was misleading. This is a mechanical rename: `SendHTML("evt", html)` → `SendData("evt", html)`.
- `EventStore.EventsAfter` now returns `([]Event, error)` instead of `[]Event` — implementations can fail (database errors, etc.) and `Replay` propagates the error instead of silently treating failures as "no events".
- `TestBroadcaster_BroadcastMany_MixedSlowFast` rewritten to use deterministic channel synchronization instead of `time.Sleep` (eliminates CI flake risk).

### Added

- `Broadcaster.BroadcastMany(msgs ...T)` — batch fan-out in a single locked pass; cheaper than looping `Broadcast` and preserves per-subscriber ordering across the batch
- `Stream.SendJSON(eventName string, v any) error` — convenience counterpart to `SendData` that JSON-marshals the payload (returns the marshal error or the write error)
- `Event.String()` — compact, human-readable representation for logging/debugging (omits empty fields); NOT the wire format
- Go `Example` functions (`ExampleWriteEvent`, `ExampleBroadcaster`, `ExampleParseEventID`) rendered in godoc
- Tests: `SendJSON` happy-path, marshal-error, and nil-value; `Send` returns error on write failure (disconnected client); concurrent `Send`+`Close` race safety; three-way `Send`+`Heartbeat`+`Close` race safety; `BroadcastMany` delivery/ordering/empty/mixed-slow-fast; `Event.String` field-omission matrix
- Benchmark: fan-out extended to 10,000 subscribers; `BenchmarkBroadcastManyVsLoop` quantifies the batch-API advantage
  - Fan-out is zero-allocation at all scales: 37 ns/op (1 sub) → 2.5 ms/op (10k subs)
  - `BroadcastMany(100 events, 1000 subs)` ≈ 1.16 ms vs 100× `Broadcast` ≈ 1.16 ms — equivalent in the uncontended case; the advantage is a single RLock pass and guaranteed per-subscriber batch ordering under contention

## [0.1.0] - 2026-07-23

### Added

- SSE wire-format serialization (`Event`, `EventID`, `WriteEvent`, `WriteHeartbeat`, `WriteRetry`) with allocation-minimized byte appends and spec-compliant multi-line `data:` splitting (LF, CRLF, and lone CR)
- `Stream`: single-connection lifecycle with mutex-guarded concurrent writes, heartbeat goroutine, `Last-Event-ID` extraction, ordered `OnDisconnect` callbacks, and request-context cancellation
- Generic `Broadcaster[T]` fan-out (over the unexported `fanOut[T]` hub): non-blocking broadcast with 64-deep subscriber buffers and drop-on-full policy, O(1) unsubscribe via channel pointer identity, subscribe/unsubscribe hooks, and graceful `Close`
- `EventStore` interface and `Replay` function for reconnection replay
- Branded `EventID` (prevents cross-assignment) with `ParseEventID` validation (rejects newlines/CR) and `MustParseEventID` for tests/constants
- `LastEventIDFromRequest` validates the `Last-Event-ID` header via `ParseEventID`, rejecting malicious values that would inject into the SSE wire format
- Fuzz tests for `WriteEvent` serialization and `ParseEventID` validation
- Integration tests with real `httptest.Server` SSE round-trip
- GitHub Actions CI workflow (test, lint, vet, coverage)
- `example/` directory with minimal SSE server (broadcaster, heartbeat, broadcast endpoint)
- `flake.nix` with hermetic build/test/lint/coverage/format automation (Go 1.26, golangci-lint, treefmt with gofumpt/goimports/golines/nixfmt)

### Changed

- `Event.Retry` type changed from `int` to `uint` — negative retry milliseconds are nonsensical; the type prevents invalid values at compile time
- `EventStore.EventsAfter` signature changed from `(string)` to `(EventID)` — callers no longer need to unwrap the branded ID with `.Get()`
- `Stream.Close` signature changed from `()` to `() error` — `Stream` now satisfies `io.Closer`
- `WriteRetry` parameter changed from `int` to `uint` for consistency with `Event.Retry`
- `Stream.Heartbeat` now delegates to `WriteHeartbeat` instead of duplicating the heartbeat frame bytes

### Fixed

- `splitLines` now handles lone CR (`\r`) as a line ending per the SSE spec, not just LF and CRLF
- `LastEventIDFromRequest` now rejects malformed `Last-Event-ID` headers containing `\n` or `\r` that would corrupt the SSE wire format (treated as empty instead of passed through)

### Security

- `LastEventIDFromRequest` validates header input with `ParseEventID`, preventing SSE wire-format injection via crafted `Last-Event-ID` headers

[Unreleased]: https://github.com/larsartmann/go-sse/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/larsartmann/go-sse/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/larsartmann/go-sse/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/larsartmann/go-sse/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/larsartmann/go-sse/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/larsartmann/go-sse/releases/tag/v0.1.0
