# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Changed

- `go-error-family` dependency bumped from `v0.8.0` to `v0.9.0`.

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

[Unreleased]: https://github.com/larsartmann/go-sse/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/larsartmann/go-sse/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/larsartmann/go-sse/releases/tag/v0.1.0
