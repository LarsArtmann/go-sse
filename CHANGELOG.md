# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## Changelog policy

What earns a line here, versus living in git history only:

- **Changelog-worthy:** every consumer- or contributor-visible change — API
  additions/changes/fixes, behavior changes, test-infrastructure gains that
  alter what CI proves (new gates, jobs, thresholds, schedules), docs that
  ship to consumers (guides, conventions), and toolchain/version bumps that
  affect how consumers build.
- **Git-history-only:** deep chore — flake/Nix plumbing, vendor-hash
  recomputes, lint-config churn, formatting sweeps. If a contributor cannot
  observe it from the outside, it does not get a line.

This ruling was formalized 2026-08-29 (decision gate D1 of the SUPERB plan)
after the cron-workflow line followed the Added precedent while the earlier
policy text said CI wiring was chore-tier; the precedent wins. It covers the
`vendorHash` split and lint-stability commits that shipped inside v0.5.1
without changelog lines (`a5ff824`, `7776bc7`).

## [Unreleased]

### Added

- `actionlint` workflow (every push/PR) with shellcheck presence proven before linting, so every `run:` block in every workflow is shellcheck-checked instead of actionlint's shellcheck integration being silently skipped; both tools added to the devShell to keep local and CI runs identical.
- The weekly flake-update workflow now closes superseded `chore/flake-update-*` PRs (branch included) before opening the new one — weekly runs can no longer pile up open PRs. Dry-run-verified on 2026-08-29 (no-drift path).
- Status-report conventions v1.1: optional `## TL;DR` section, cover-line scope rules for cross-repo sessions, and an explicit note that `docs/status/AGENTS.md` specializes the global report format for this repo.
- `docs/status/_template.md` — copy-pasteable report skeleton (title, preamble, coverage-delta line, a–g sections), linked from the conventions and the root AGENTS.md pointer.
- Changelog policy formalized as a living top-level section (SUPERB decision gate D1): consumer/contributor-visible changes earn lines — including CI gates, jobs, thresholds, and schedules — while deep chore (flake plumbing, vendor-hash recomputes, lint-config churn) stays git-history-only.

### Fixed

- Nothing yet.

## [ssetest 0.3.0] - 2026-08-29

### Added

- ssetest `RequireDataJSON(tb, evt, want)` — unmarshal-and-compare assertion for JSON-payload events: structural compare (key order and whitespace don't matter), type inferred from `want`, fatal-with-payload on invalid JSON.
- `FuzzSplitSSELines` — direct fuzz target for the reader's line splitter, property-pinned against an independent spec § 9.2.5 reference model (CR/LF/CRLF terminators, CRLF-is-one, no trailing empty line), with 19 committed corpus entries.
- BOM boundary matrix test: leading, double, and mid-stream BOM driven through every chunk size 1–7 (the probe's own `ReadFull` states).
- E2E sticky-ID reconnect test (`TestE2E_StickyIDSurvivesReconnect`): wire `id:` → parser sticky state → `Last-Event-ID` echo → replay → post-replay events inherit the last seen ID.
- ssetest fuzz corpus committed: 156 regression seeds across `FuzzReadEvents` (51), `FuzzWriteReadRoundTrip` (86), `FuzzSplitSSELines` (19), including the `"0data: hello\n\n"` crasher that exposed the substring-assertion bug.

### Changed

- Requires `go-sse v0.6.0` (paired release; the previous pin was v0.5.1).
- ssetest `Collect*` helpers factor response-body closing into `closeBody`, making the `Close`-error branch unit-testable with an erroring `io.ReadCloser` fake (both branches now covered).
- ssetest module `go` directive aligned with the root module (1.26.7).
- ssetest coverage 95.3% → 97.2%.

## [0.6.0] - 2026-08-29

### Added

- `safeDropCall` panic recovery for the fan-out drop callback: a panicking `onDrop` no longer crashes `Broadcast`/`BroadcastMany` on the broadcasting goroutine — symmetric with the existing predicate-recovery contract. Pinned by `TestOnDrop_PanickingCallbackDoesNotCrashBroadcast(Many)`.
- Wire round-trip tests for `KeyedLines`/`SendKeyed` (`keyed_wire_test.go`): every keyed line survives `WriteEvent`/`Stream.Send` verbatim; rejoined data lines reconstruct the exact `KeyedLines` result; `JoinLines` composition order pinned.
- `scripts/verify.sh` — one-command pre-push gate (treefmt + vet + lint + race test + `nix flake check`; `--fast` skips the flake check). Direnv-independent: exports `GOWORK=off`/`GOEXPERIMENT=jsonv2` itself.
- CI: examples build + `templ generate -check` staleness job; `nix flake check` hermetic-gate job; fuzz job extended from 3 to 6 targets (`FuzzKeyedLines`, `FuzzWriteReadRoundTrip`, `FuzzSplitSSELines`); `govulncheck` pinned to v1.7.0 (was `@latest`).
- `coverage-gate` flake app now enforces an ssetest threshold (95%) alongside the library's 90%, and is hermetic (declared runtime inputs, own `GOWORK`/`GOEXPERIMENT` exports).
- `docs/guides/reconnection-and-retry.md` — the 5-layer reconnection model (browser `EventSource` → `retry:` hint → `Last-Event-ID`+`Replay` → `Heartbeat` → `Shutdown` drain).
- CONTRIBUTING release checklist: CHANGELOG cut, FEATURES/ROADMAP refresh, worktree tag validation, proxy verification, `gh release create`, dual-module tagging rules.
- Tests for `eventBrand.Name()` and its `brandid.BrandNamer` wiring (0% covered since `TestEventID_StringIncludesBrandName` was deleted 2026-07-27), and for `OnDrop(nil)`/`WithOnDrop(nil)` clear-callback behavior.
- Weekly `nix flake update` CI schedule (`.github/workflows/flake-update.yml`): Mondays 04:00 UTC it bumps `flake.lock`, runs the full `nix flake check` gate on the bumped inputs in-workflow (GITHUB_TOKEN-opened PRs never trigger CI, so the gate cannot live on the PR), and opens a review PR — input drift is caught within a week instead of at the next manual bump.
- `docs/status/AGENTS.md` — status-report conventions: file naming, the a–g section skeleton, harvest rules, and a mandatory coverage-delta line (`- cover: library …, ssetest …`, measured per session). A coverage regression like 2026-07-27's 100% → 99.5% can no longer ship unnoticed.
- AGENTS.md Gotcha: auto-git daemon commit messages are plausible-but-false for deletions (`38e79aa` says "expand test coverage" for a coverage-reducing deletion) — sanity-check them after deletion-heavy sessions.

### Fixed

- `FuzzKeyedLines` property now applies only to line-safe keys: the "every output line starts with the key" invariant is meaningless when the key itself contains CR/LF (out of the keyed-data-line contract). Found by CI fuzzing after the property shipped; the crasher (`key="\n"`) is pinned as a corpus entry.
- CI lint aligned with the flake's golangci-lint (v2.13.1, was v2.12.2): the version skew let a `goconst` counting difference between 2.12 and 2.13 pass locally and fail CI. The `"SSEEvent"` literal is now a named constant in `event.go` and the brand test.

### Changed

- Go toolchain bumped to 1.26.7; Nix flake inputs refreshed (`713db38`). No library code changes.
- `WithOnDrop`/`OnDrop` doc comments now state the re-entrancy constraint (the callback runs under the fan-out read lock; calling back into the broadcaster deadlocks) and the panic-recovery contract; `Broadcast`/`BroadcastMany` cross-reference the drop-observability hook.
- Library coverage 99.3%; `eventBrand.Name()` 0% → 100%.
- Flake `systems` narrowed to `x86_64-linux`, `aarch64-linux`, `aarch64-darwin`: Nixpkgs 26.11 dropped x86_64-darwin, so its derivations no longer evaluate; `nix flake check --all-systems` is green on the declared systems. Pure Go — excluded hosts still build from source.
- Inferable generic type arguments removed from test call sites (`NewBroadcaster(WithBufferSize[int](1))`, `WithOnDrop(func…)`); required ones kept (`NewBroadcaster[int]()`, `WithBufferSize[int](…)`).

## [ssetest 0.2.0] - 2026-08-22

### Added

- `StreamReader` type and `NewStreamReader` constructor for reading SSE events one at a time from a live stream. Unlike calling `ReadNEvents` repeatedly (which creates and discards a new `bufio.Scanner` per call, losing buffered data), `StreamReader` wraps a single scanner across all `Next()` calls — the correct API for test patterns that interleave reading SSE events with triggering actions (POST, mutate state, then read the next event).
- `MustReadNextEvent(tb testing.TB, sr *StreamReader) Event` — fatal helper for `StreamReader.Next`, accepts `testing.TB` so it works with `*testing.T`, `*testing.B`, and `GinkgoT()`.

### Changed

- `ReadNEvents` and `MustReadNEvents` doc comments now warn that each call creates a new scanner and recommend `StreamReader` for repeated reads on the same live stream.

## [ssetest 0.1.0] - 2026-08-22

First release of the consumer test-helper module (`github.com/larsartmann/go-sse/ssetest`). Import it in tests only — because it is its own module, `testing` never leaks into consumer production builds.

### Added

- Spec-based conformance test suite pinning go-sse to the WHATWG HTML Living Standard § 9.2 (Server-Sent Events):
  - `wpt_format_corpus_test.go` — the official Web Platform Tests (WPT) `eventsource/format-*.any.js` corpus transcribed into executable Go tests (17 WPT vectors, 4 spec § 9.2.6 example streams, 8 Chromium `event_source_parser_test.cc` unit cases, each with its upstream citation). The official browser suite now runs in our CI forever.
  - `chunk_boundary_test.go` — the entire corpus re-parsed through readers delivering 1–4096 byte chunks (Chromium's `EnqueueOneByOne` trick), proving parse results are independent of TCP chunking, including CRLF pairs and BOM bytes split across reads.
  - `roundtrip_test.go` — writer/reader round-trip table plus `FuzzWriteReadRoundTrip` property: everything `sse.WriteEvent` writes reads back as the identical observable event (modulo the spec-mandated CR/CRLF→LF normalization and the structural trailing newline).
  - `FuzzReadEvents` extended with 25 conformance seeds (BOM, NUL id, sticky id, lone CR, EOF-discard shapes) and a byte-by-byte chunk-invariance property.
- Consumer test helpers: SSE wire-format parser (`ReadEvents`/`ReadNEvents` with dataless-frame suppression per the SSE spec), end-to-end `Collect*` helpers that drive a real HTTP server (`Collect`, `CollectPost`, `CollectWithRequest`, `CollectN`, `CollectWithTimeout`), request options (`WithPath`, `WithHeader`, `WithLastEventID` for reconnect/replay testing), `Require*` assertions, `FindByType`/`FilterByType` search, and `EventsString` debugging. All helpers accept `testing.TB` (works with `*testing.T`, `*testing.B`, and Ginkgo's `GinkgoT()`). 94.6% statement coverage (0 erraudit violations with `--enforce-go-error-family`); includes dogfood E2E tests over `Stream`, `Broadcaster`, `Replay`, and heartbeats.
- `flake.nix` build/test/lint/coverage apps now cover the `ssetest/` module alongside the root module, and `nix flake check` builds a hermetic compile+test derivation (`checks.build-ssetest`) for it; CI runs test/vet/lint/coverage/vulncheck/fuzz for both modules.

### Fixed

- `ReadEvents`/`ReadNEvents` parser brought into spec conformance (WHATWG HTML § 9.2.6; verified against WPT). Six deviations corrected, all behavioral (no API signature changes):
  - **Lone CR is a line terminator** (§ 9.2.5 `end-of-line = cr lf / cr / lf`). Previously only LF terminated lines (with CRLF tolerated), so CR-terminated streams — which every browser parses — mis-parsed. The reader now uses a dedicated CR/LF/CRLF split function.
  - **An incomplete final frame at EOF is discarded**, per "Once the end of the file is reached, any pending data must be discarded" (WPT `format-data-before-final-empty-line`). Previously the trailing frame was leniently dispatched.
  - **Exactly one leading UTF-8 BOM is stripped** (WPT `format-bom`, `format-bom-2`). Previously a BOM poisoned the first field name. A second, mid-stream BOM is data and still poisons the field name, exactly as in browsers.
  - **An `id:` field containing U+0000 NULL is ignored** (WPT `format-field-id-null`), instead of being accepted as the event ID.
  - **The last event ID is sticky** (§ 9.2.6 dispatch step 1; Chromium `LastEventIdShouldNotBeReset`): `Event.ID` now reports the most recent `id:` value in effect at dispatch time — it persists across frames that don't restate it, an id-only frame's value reaches the next dispatched event, and an empty `id:` resets it to `""`. Previously IDs were per-frame.
  - The `retry:` reconnection time is likewise connection-level state (Chromium `RetryTakesEffectEvenWhenNotDispatching`): a `retry:` line updates it even in frames that never dispatch, invalid values never reset a previously set value, and parsing widened to 64-bit width before assignment.

### Changed

- **`Event.ID` and `Event.Retry` semantics are "value in effect at dispatch"** (sticky, per spec) instead of "value restated in this frame". Tests that asserted `ID == ""` for events following an `id:` frame, or `Retry == 0` for events following a `retry:` frame, must be updated. The struct fields keep their names and types.
- Reader internals restructured into a `streamParser` with explicit per-frame vs connection-level state (the spec's event type/data buffers vs last-event-ID buffer/reconnection time). No public API change.

## [0.5.1] - 2026-08-22

### Added

- `event_spec_test.go` — writer golden vectors transcribed from spec § 9.2.6 (stock ticker, space-after-colon note, field order, empty-data dispatch, CR/CRLF line splitting, heartbeat comment form) and `ParseEventID` value-space tests.

### Fixed

- `sse.ParseEventID` now rejects U+0000 NULL in addition to LF/CR, completing the spec § 9.2.4 Last-Event-ID value space (browsers ignore id fields containing NUL, so accepting one produced IDs no browser would ever echo back). Error message updated to name all three forbidden character classes.
- `TestSubscribeFilter_ConcurrentRace` no longer flakes under `-race` on contended CI runners: the persistent subscriber's buffer is now sized to hold the entire burst (`WithBufferSize`), turning the flaky `>= 100` lower-bound assertion into an exact, deterministic no-loss check (`received == sent`), which is also a stronger guarantee.

## [0.5.0] - 2026-08-13

### Added

- `WithOnDrop[T](fn func(T)) Option[T]` — constructor option that registers a callback invoked each time a message is dropped because a subscriber's buffer is full. The callback receives the dropped message and fires once per full subscriber per broadcast. Runs on the broadcasting goroutine inside the fan-out read lock; must be fast and non-blocking (atomic increment, buffered channel send). Pass nil to clear.
- `Broadcaster.OnDrop(fn func(T))` — runtime setter equivalent to the constructor option, for consumers that wire callbacks after construction (e.g. metrics registry handed in after `NewBroadcaster`). Pass nil to clear.
- `JoinLines(lines ...string) string` — joins variadic arguments with `\n`, producing the `Event.Data` string for multi-line SSE events. Composes with `KeyedLines` for multi-key protocols like DataStar (e.g., `JoinLines("selector #feed", "mode inner", KeyedLines("elements", html))`).
- DataStar example overhauled into a full activity feed showcasing every go-sse feature: `Broadcaster` fan-out (multiple tabs), `SubscriberCount` (live counter via `OnSubscribe`/`OnUnsubscribe`), `SubscribeFilter` (`?filter=alerts`), `EventStore` + `Replay` (close/reopen tab), and `Heartbeat` (proxy keep-alive). Background producer emits events every 2s.
- HTMX example (`example/htmx/`) — progress-bar demo using plain `stream.Send` to stream HTML fragments; HTMX SSE extension swaps them into the DOM.
- `example/README.md` — detailed DataStar vs HTMX comparison (mechanism, payload size, granularity, bundle size, trade-offs).
- DataStar example migrated to `a-h/templ` type-safe HTML templates with embedded static assets (no CDN). Generated `*_templ.go` checked into git.
- Theme toggle (dark/light via `prefers-color-scheme` + manual `data-theme`), pause control, CORS headers, and live event counter in the DataStar example.
- `flake.nix` coverage-gate app enforcing 90% library coverage threshold (`nix run .#coverage`).
- Vendored client bundles: DataStar v1.0.2, HTMX 2.0.4 + htmx-ext-sse 2.2.4 (no CDN dependencies).
- `docs/performance/scale-profile.md` — memory and latency characterization at 100/1k/10k subscribers (~4 KiB each, buffer-dominated). Conclusion: default 64-buffer and non-blocking drop policy are well-calibrated.
- `BenchmarkMemoryPerSubscriber` — steady-state heap memory per subscriber at 100/1k/10k subscribers.
- `BenchmarkWriteEvent` — measures `WriteEvent` allocation behavior.
- `ExampleReplayFiltered` — godoc example demonstrating predicate-based reconnection replay.
- `TestSubscribeFilter_DropPolicyRespected`, `TestSubscribeFilter_BroadcastManyMixedSubscribers`, `TestWithOnDrop_FiresWhenBufferFull`, `TestWithOnDrop_FiresPerSubscriber`, `TestWithOnDrop_BroadcastMany`, `TestOnDrop_RuntimeRegistration` — new tests pinning drop semantics, filter+drop interaction, and onDrop callback behavior.

### Changed

- Error wrapping improved across the library: `WriteEvent`, `Send`, `Replay`, and `Shutdown` now use `errorfamily.Wrapf` with stable severity categories (`Transient`, `Rejection`) and machine-readable codes, enabling consumers to classify errors programmatically.
- Internal refactors for clarity: subscription helpers extracted, drain-wait logic moved to `waitForDrain` helper, subscriber snapshot uses `slices.Collect(maps.Values(...))`, event-writing loop optimized with direct byte appends.
- Go 1.26 idioms adopted throughout: integer range loops (`for i := range len(s)`), `sync.WaitGroup.Go`, `slices`/`maps` packages.
- `Shutdown` error context enriched: wraps the underlying `context.Canceled`/`context.DeadlineExceeded` with code `sse.shutdown_drain_deadline_exceeded` while preserving `errors.Is` compatibility.
- DataStar example split across four files (`main.go`, `store.go`, `producer.go`, `handlers.go`) for readability.

### Fixed

- `example/datastar/main.go` progress bar corrected: `data-bind:style` → `data-style:width="$progress + '%'"`. `data-bind` is DataStar's form-element two-way binding; CSS styles use `data-style`.
- `NewBroadcaster` removed an unnecessary explicit type argument (`newFanOut[T]` → `newFanOut`) flagged by gopls `infertypeargs`.
- `.envrc` now exports `GOEXPERIMENT=jsonv2` alongside `GOWORK=off`, so `buildflow`, `gopls`, and direct `go` invocations launched outside the Nix devShell inherit the flag via direnv.
- README.md and doc.go predicate panic-policy documentation corrected: both now state that panicking predicates are recovered and treated as non-matches (matching v0.4.0 code behavior).
- `example/datastar/main.go` CDN URL corrected: Cloudflare email-protection placeholder leaked into the URL → `datastar@1.0.2`. The previous URL returned `400 Bad Request` from jsdelivr. (Now moot — assets are vendored.)
- `TestSubscribeFilter_ConcurrentRace` threshold lowered from 500 to 100 to eliminate CI flakiness on contended runners while still proving non-blocking delivery.

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

[Unreleased]: https://github.com/larsartmann/go-sse/compare/ssetest/v0.2.0...HEAD
[ssetest 0.2.0]: https://github.com/larsartmann/go-sse/compare/ssetest/v0.1.0...ssetest/v0.2.0
[ssetest 0.1.0]: https://github.com/larsartmann/go-sse/compare/v0.5.1...ssetest/v0.1.0
[0.5.1]: https://github.com/larsartmann/go-sse/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/larsartmann/go-sse/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/larsartmann/go-sse/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/larsartmann/go-sse/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/larsartmann/go-sse/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/larsartmann/go-sse/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/larsartmann/go-sse/releases/tag/v0.1.0
