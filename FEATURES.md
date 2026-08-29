# Features

Honest inventory of what `go-sse` does and its real status. Verified by running `go test ./... -race -count=1` (all passing, 99.3% statement coverage of the core package) and `golangci-lint run ./...` (0 issues). See test inventory with `go test ./... -v`.

## Status vocabulary

Only 4 statuses are used. Non-goals (below) are listed outside this system because they have no code to assess.

| Status               | Meaning                                                      |
| -------------------- | ------------------------------------------------------------ |
| FULLY_FUNCTIONAL     | Code present AND working (tests pass or exercised).          |
| PARTIALLY_FUNCTIONAL | Ships but has known gaps, edge-case bugs, or missing pieces. |
| BROKEN               | Code exists but does not work / is disabled / fails.         |
| PLANNED              | Designed or documented but **no code exists yet**.           |

## Wire format & serialization

| Feature                                                                                 | Status           | Evidence                                                                                                     |
| --------------------------------------------------------------------------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------ |
| SSE event serialization (`event`/`data`/`id`/`retry`)                                   | FULLY_FUNCTIONAL | `WriteEvent` in `event.go`; `event_test.go`                                                                  |
| Content-type + event-name constants (`ContentType`, `EventConnected`, `EventHeartbeat`) | FULLY_FUNCTIONAL | `constants.go`; used by `example/server.go`                                                                  |
| Multi-line `data:` splitting (LF, CRLF, and lone CR per spec)                           | FULLY_FUNCTIONAL | `splitLines` in `event.go`; `TestWriteEvent_CRLFInData`, `TestWriteEvent_LoneCarriageReturn`                 |
| Allocation-minimized writer (byte appends, no `fmt` on hot path)                        | FULLY_FUNCTIONAL | `WriteEvent` in `event.go`                                                                                   |
| Heartbeat comment frames                                                                | FULLY_FUNCTIONAL | `WriteHeartbeat` in `event.go`; `TestWriteHeartbeat`                                                         |
| Retry directive (`uint`, rejects negative at compile time)                              | FULLY_FUNCTIONAL | `WriteRetry` in `event.go`; `TestWriteRetry`, `TestWriteEvent_Retry`                                         |
| `Event.String()` debug representation (omits empty fields)                              | FULLY_FUNCTIONAL | `Event.String` in `event.go`; `TestEvent_String`                                                             |
| `KeyedLines` (keyed data-line helper for DataStar and similar protocols)                | FULLY_FUNCTIONAL | `KeyedLines` in `event.go`; `TestKeyedLines_*`, `ExampleKeyedLines`, `FuzzKeyedLines`, `BenchmarkKeyedLines` |
| `WriteKeyedLines` (wire-only single-key helper, no `net/http`)                          | FULLY_FUNCTIONAL | `WriteKeyedLines` in `event.go`; `TestWriteKeyedLines_*`                                                     |

## Connection management (`Stream`)

| Feature                                                                    | Status           | Evidence                                                                                                                    |
| -------------------------------------------------------------------------- | ---------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Stream lifecycle: SSE headers + `200 OK` on creation                       | FULLY_FUNCTIONAL | `NewStream` in `stream.go`; `stream_test.go`                                                                                |
| `SetHeaders` (set SSE headers without writing status, for custom handlers) | FULLY_FUNCTIONAL | `SetHeaders` in `stream.go`; used internally by `NewStream`                                                                 |
| Send event + flush                                                         | FULLY_FUNCTIONAL | `Stream.Send` in `stream.go`; `TestStream_Send`                                                                             |
| Concurrent-safe writes (mutex-serialized `Send`/`Heartbeat`/`Close`)       | FULLY_FUNCTIONAL | `Stream.mu` in `stream.go`; `TestStream_SendHeartbeatRaceSafety`, `TestStream_SendHeartbeatCloseRace` (race-tested)         |
| Heartbeat goroutine (proxy keep-alive)                                     | FULLY_FUNCTIONAL | `Stream.Heartbeat` in `stream.go`; `TestStream_Heartbeat`                                                                   |
| `Last-Event-ID` header extraction (validated via `ParseEventID`)           | FULLY_FUNCTIONAL | `LastEventIDFromRequest` in `stream.go`; `TestLastEventIDFromRequest_MaliciousInputTreatedAsEmpty`                          |
| `OnDisconnect` callbacks (ordered)                                         | FULLY_FUNCTIONAL | `Stream.OnDisconnect` in `stream.go`; `TestStream_OnDisconnectMultipleInOrder`                                              |
| `SendData` convenience                                                     | FULLY_FUNCTIONAL | `Stream.SendData` in `stream.go`; `TestStream_SendData`                                                                     |
| `SendJSON` convenience (marshal + send)                                    | FULLY_FUNCTIONAL | `Stream.SendJSON` in `stream.go`; `TestStream_SendJSON`, `TestStream_SendJSON_MarshalError`, `TestStream_SendJSON_NilValue` |
| `SendLines` convenience (multi-line data lines, DataStar-compatible)       | FULLY_FUNCTIONAL | `Stream.SendLines` in `stream.go`; `TestStream_SendLines`                                                                   |
| `SendKeyed` convenience (single-key DataStar pattern)                      | FULLY_FUNCTIONAL | `Stream.SendKeyed` in `stream.go`; `TestStream_SendKeyed`, `TestStream_SendKeyed_MultiLine`                                 |
| `Send` error on write failure (disconnected client)                        | FULLY_FUNCTIONAL | `Stream.Send` in `stream.go`; `TestStream_SendReturnsErrorOnWriteFailure`                                                   |
| Concurrent `Send`+`Close` race safety                                      | FULLY_FUNCTIONAL | `TestStream_SendCloseRace`, `TestStream_SendHeartbeatCloseRace` (three-way race-tested)                                     |
| Request-context cancellation                                               | FULLY_FUNCTIONAL | `Stream.Context` in `stream.go`; `TestStream_ContextCancellation`                                                           |
| `io.Closer` interface compliance                                           | FULLY_FUNCTIONAL | `Stream.Close` in `stream.go`; `TestStream_DoubleCloseSafety`                                                               |

## Fan-out (`Broadcaster[T]` / `fanOut[T]`)

| Feature                                              | Status           | Evidence                                                                                                                                                                                                               |
| ---------------------------------------------------- | ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Generic fan-out over any message type                | FULLY_FUNCTIONAL | `Broadcaster[T]` in `broadcaster.go`; `TestBroadcaster_GenericWithString`                                                                                                                                              |
| Subscribe / Unsubscribe                              | FULLY_FUNCTIONAL | `fanOut.Subscribe`, `fanOut.Unsubscribe` in `fanout.go`; `TestBroadcaster_Subscribe`                                                                                                                                   |
| O(1) unsubscribe via channel pointer identity        | FULLY_FUNCTIONAL | `channelPtr` in `fanout.go`; `TestBroadcaster_BroadcastUnsubscribeRace`                                                                                                                                                |
| Non-blocking broadcast (drops to slow consumers)     | FULLY_FUNCTIONAL | `fanOut.Broadcast` in `fanout.go`; `TestBroadcaster_DropsOnFullBuffer`                                                                                                                                                 |
| Predicate-based subscription (`SubscribeFilter`)     | FULLY_FUNCTIONAL | `fanOut.SubscribeFilter` in `fanout.go`; `TestSubscribeFilter_*` (11 tests, including panic recovery, Shutdown drain, and drop policy)                                                                                 |
| Batch broadcast (`BroadcastMany`, single lock pass)  | FULLY_FUNCTIONAL | `fanOut.BroadcastMany` in `fanout.go`; `TestBroadcaster_BroadcastMany`, `_PreservesOrder`, `_MixedSlowFast`                                                                                                            |
| Broadcast after Close is safe (no panic)             | FULLY_FUNCTIONAL | `TestBroadcaster_BroadcastAfterClose`                                                                                                                                                                                  |
| `OnSubscribe` / `OnUnsubscribe` hooks                | FULLY_FUNCTIONAL | `fanOut.OnSubscribe`, `fanOut.OnUnsubscribe` in `fanout.go`; `TestBroadcaster_OnSubscribeHook`                                                                                                                         |
| Graceful `Close` (closes all channels)               | FULLY_FUNCTIONAL | `fanOut.Close` in `fanout.go`; `TestBroadcaster_Close`                                                                                                                                                                 |
| Subscribe-after-close returns closed channel (no-op) | FULLY_FUNCTIONAL | `fanOut.Subscribe` in `fanout.go`; `TestBroadcaster_SubscribeAfterClose`                                                                                                                                               |
| Graceful `Shutdown(ctx)` drain with deadline         | FULLY_FUNCTIONAL | `fanOut.Shutdown` in `fanout.go`; `TestBroadcaster_Shutdown_DrainsAllSubscribers`, `_Empty`, `_ContextCancel`, `_RejectsNewSubscribersWhileDraining`, `_Idempotent`, `_AfterCloseIsNoop`, `_ConcurrentWithUnsubscribe` |
| `Broadcaster.Health()` structured status snapshot    | FULLY_FUNCTIONAL | `BroadcasterHealth` in `fanout.go`; `TestBroadcaster_Health_InitialState`, `_DuringOperation`, `_AfterClose`, `_ReportsBufferSize`                                                                                     |
| Configurable subscriber buffer (`WithBufferSize`)    | FULLY_FUNCTIONAL | `Option[T]`, `WithBufferSize[T]` in `fanout.go`; `TestBroadcaster_WithBufferSize_AppliesToNewSubscribers`, `_NonPositiveIsIgnored`                                                                                     |
| Concurrent-safety (race-tested)                      | FULLY_FUNCTIONAL | `TestBroadcaster_ConcurrentSafety`, `TestBroadcaster_ConcurrentHookCount`                                                                                                                                              |

## Reconnection replay

| Feature                                                        | Status           | Evidence                                                                                                                      |
| -------------------------------------------------------------- | ---------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `EventStore` interface (consumer-implemented, takes `EventID`) | FULLY_FUNCTIONAL | `replay.go`                                                                                                                   |
| `Replay` function (replays missed events after a given ID)     | FULLY_FUNCTIONAL | `Replay` in `replay.go`; `TestReplay_AfterGivenID`, `TestReplay_NoLastID`                                                     |
| `FilteredEventStore` interface (predicate push-down to store)  | FULLY_FUNCTIONAL | `FilteredEventStore` in `replay.go`; `TestReplayFiltered_FilteredEventStorePath`                                              |
| `ReplayFiltered` (replays only events matching a predicate)    | FULLY_FUNCTIONAL | `ReplayFiltered` in `replay.go`; `TestReplayFiltered_*` (7 tests, including fallback panic recovery), `ExampleReplayFiltered` |
| Write-failure error propagation                                | FULLY_FUNCTIONAL | `Replay` in `replay.go`; `TestReplay_WriteError`                                                                              |
| Store-error propagation (`EventsAfter` returns error)          | FULLY_FUNCTIONAL | `Replay` in `replay.go`; `TestReplay_StoreError`                                                                              |

## Type safety

| Feature                                                             | Status           | Evidence                                                                                     |
| ------------------------------------------------------------------- | ---------------- | -------------------------------------------------------------------------------------------- |
| Branded `EventID` (prevents cross-assignment with other string IDs) | FULLY_FUNCTIONAL | `event.go`; `TestEventID_IsZero`                                                             |
| `ParseEventID` (rejects `\n`/`\r` that corrupt the wire format)     | FULLY_FUNCTIONAL | `ParseEventID` in `event.go`; `TestParseEventID_RejectsNewlines`, `TestParseEventID_Unicode` |
| `MustParseEventID` (panicking variant for tests/constants)          | FULLY_FUNCTIONAL | `MustParseEventID` in `event.go`; `TestMustParseEventID_Panics`                              |

## Testing infrastructure

| Feature               | Status           | Evidence                                                                                                                                                                                                                                                                                                                                     |
| --------------------- | ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Fuzz tests            | FULLY_FUNCTIONAL | `FuzzWriteEvent`, `FuzzParseEventID`, `FuzzKeyedLines` in `fuzz_test.go`; plus 3 ssetest targets (`FuzzReadEvents`, `FuzzWriteReadRoundTrip`, `FuzzSplitSSELines`) — all 6 run 1m each in CI with committed regression corpora                                                                     |
| Integration tests     | FULLY_FUNCTIONAL | `TestIntegration_DirectSendAndHeaders`, `_BroadcasterFanOut`, `_HeartbeatDelivery`, `_LastEventIDReconnectionReplay`, `_DataStarWireFormat`, `_SubscribeFilter`, `_ReplayFiltered` in `integration_test.go`                                                                                                                                  |
| Race detector tests   | FULLY_FUNCTIONAL | `TestStream_SendHeartbeatRaceSafety`, `TestStream_SendCloseRace`, `TestStream_SendHeartbeatCloseRace`, `TestBroadcaster_BroadcastUnsubscribeRace`, `TestSubscribeFilter_ConcurrentRace`                                                                                                                                                      |
| Example tests (godoc) | FULLY_FUNCTIONAL | `ExampleWriteEvent`, `ExampleBroadcaster`, `ExampleBroadcaster_SubscribeFilter`, `ExampleParseEventID`, `ExampleKeyedLines`, `ExampleReplayFiltered` in `example_test.go`                                                                                                                                                                    |
| Benchmarks            | FULLY_FUNCTIONAL | `BenchmarkBroadcasterFanOut` (1–10k subs), `BenchmarkSubscribeUnsubscribe`, `BenchmarkBroadcastManyVsLoop` in `broadcaster_test.go`; `BenchmarkKeyedLines` in `event_test.go`; `BenchmarkSubscribeFilter_PredicateOverhead` in `filter_test.go`; `BenchmarkMemoryPerSubscriber` (per-subscriber heap at 100/1k/10k) in `broadcaster_test.go` |
| CI pipeline           | FULLY_FUNCTIONAL | `.github/workflows/ci.yml`                                                                                                                                                                                                                                                                                                                   |

## Consumer test helpers (`ssetest/`)

Separate Go module (`github.com/larsartmann/go-sse/ssetest`), so `testing` never leaks into consumer production builds. 97.2% statement coverage (2026-08-29); 0 erraudit violations with `--enforce-go-error-family`.

| Feature                                                                                     | Status           | Evidence                                                                                                                                                                                        |
| ------------------------------------------------------------------------------------------- | ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| SSE wire-format parser, WHATWG § 9.2.6-conformant                                           | FULLY_FUNCTIONAL | `ReadEvents`, `ReadNEvents`, `MustRead*` in `ssetest/reader.go` (custom CR/LF/CRLF split, BOM strip-once, sticky id/retry, NUL-id ignore, EOF discard); `reader_test.go`, `BenchmarkReadEvents` |
| `StreamReader` (one-at-a-time reads on a live stream, single scanner across `Next()` calls) | FULLY_FUNCTIONAL | `StreamReader`, `NewStreamReader`, `Next` in `ssetest/reader.go` (added v0.2.0 — unlike `ReadNEvents`, no buffered data is lost between calls); `TestStreamReader_*` in `reader_test.go`        |
| `MustReadNextEvent` fatal helper (`testing.TB`)                                             | FULLY_FUNCTIONAL | `MustReadNextEvent` in `ssetest/reader.go`; `TestMustReadNextEvent`                                                                                                                             |
| WPT conformance corpus (official browser suite as Go tests)                                 | FULLY_FUNCTIONAL | `wpt_format_corpus_test.go` — 17 WPT `eventsource/format-*` vectors, 4 spec § 9.2.6 example streams, 8 Chromium `event_source_parser_test.cc` cases, each with upstream citation                |
| Chunk-boundary independence (TCP chunking never changes parse)                              | FULLY_FUNCTIONAL | `chunk_boundary_test.go` — full corpus re-parsed through 1–4096 byte chunked readers; BOM boundary matrix (leading/double/mid-stream BOM × every chunk size 1–7); byte-by-byte property in `FuzzReadEvents`                      |
| Writer/reader round-trip identity                                                           | FULLY_FUNCTIONAL | `roundtrip_test.go` table + `FuzzWriteReadRoundTrip` (root `sse.Event` → `WriteEvent` → `ReadEvents` → identity, CR/CRLF→LF normalization pinned)                                               |
| Dataless-frame suppression (heartbeats/id-only frames never dispatch)                       | FULLY_FUNCTIONAL | `streamParser.dispatchFrame` in `ssetest/reader.go`; `TestReadEvents_DatalessFramesNeverDispatch`, `FuzzReadEvents`                                                                             |
| End-to-end Collect helpers (real HTTP server via httptest)                                  | FULLY_FUNCTIONAL | `Collect`, `CollectPost`, `CollectWithRequest`, `CollectN`, `CollectWithTimeout` in `ssetest/collect.go`                                                                                        |
| Request options (path/query, headers, Last-Event-ID)                                        | FULLY_FUNCTIONAL | `WithPath`, `WithHeader`, `WithLastEventID` in `ssetest/options.go`; `options_test.go`                                                                                                          |
| `Require*` assertions (count, type, data, ID, retry)                                        | FULLY_FUNCTIONAL | `ssetest/assert.go`; `assert_test.go` (failure paths via recordingTB)                                                                                                                           |
| `RequireDataJSON` (unmarshal-and-compare for JSON payloads)                                 | FULLY_FUNCTIONAL | `RequireDataJSON` in `ssetest/assert.go`; `TestRequireDataJSON_*` in `assert_test.go` (added 2026-08-29)                                                                                        |
| Event search without index math                                                             | FULLY_FUNCTIONAL | `FindByType`, `FilterByType` in `ssetest/search.go`                                                                                                                                             |
| `testing.TB` compatibility (T, B, GinkgoT)                                                  | FULLY_FUNCTIONAL | All public helpers take `tb testing.TB`; `TestHelpers_AcceptTestingB`                                                                                                                           |
| Dogfood E2E over real go-sse types                                                          | FULLY_FUNCTIONAL | `e2e_test.go` (Stream round-trip, Broadcaster fan-out, Replay+LastEventID, heartbeat invisibility, timeout read, sticky-ID reconnect end-to-end)                                                |
| Hermetic Nix check for the nested module                                                    | FULLY_FUNCTIONAL | `hermeticCheckSsetest` (`checks.build-ssetest`) in `flake.nix`; `preBuild = "cd ssetest"` bridges buildGoModule's root-module assumption                                                        |
| Debug rendering for failure messages                                                        | FULLY_FUNCTIONAL | `Event.String`, `EventsString` in `ssetest/event.go`; `example_test.go` (godoc examples with output)                                                                                            |

Example coverage note: the runnable `example/` packages are measured in the repo-wide `go test ./... -cover` total (root 98.9%, `example/datastar` 46.3%, `example/server.go` and `example/htmx` 0%) but are excluded from the library coverage gate, which measures the `sse` package only.

## Explicit non-goals

These are deliberately out of scope. They are documented here so contributors
do not mistake their absence for a gap.

- WebSocket support
- CQRS dispatch hooks
- Dashboard server, routes, or HTML templates
- Event bus integration
- Any opinion about payload format (strings, JSON, HTML fragments are all fine)
- `Broadcaster.ServeSSE` convenience handler (would bake in heartbeat, replay, and event-loop opinions; the `example/` package shows the canonical pattern)

## Dependencies

Two same-author utility modules (the only `require` entries in `go.mod`):

- `github.com/larsartmann/go-branded-id` — phantom-type branded IDs (`EventID`)
- `github.com/larsartmann/go-error-family` — structured error wrapping

Requires Go 1.26.7+ (both modules' `go` directives) with `GOEXPERIMENT=jsonv2` (transitive, via `go-branded-id`).
