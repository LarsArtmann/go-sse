# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

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
