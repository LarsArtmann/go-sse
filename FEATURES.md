# Features

Honest inventory of what `go-sse` does and its real status. Verified by running `go test ./... -race -count=1` (all passing, 97.9% coverage) and `golangci-lint run ./...` (0 issues). See test inventory with `go test ./... -v`.

## Status vocabulary

Only 4 statuses are used. Non-goals (below) are listed outside this system because they have no code to assess.

| Status               | Meaning                                                      |
| -------------------- | ------------------------------------------------------------ |
| FULLY_FUNCTIONAL     | Code present AND working (tests pass or exercised).          |
| PARTIALLY_FUNCTIONAL | Ships but has known gaps, edge-case bugs, or missing pieces. |
| BROKEN               | Code exists but does not work / is disabled / fails.         |
| PLANNED              | Designed or documented but **no code exists yet**.           |

## Wire format & serialization

| Feature                                                          | Status           | Evidence                                                                                     |
| ---------------------------------------------------------------- | ---------------- | -------------------------------------------------------------------------------------------- |
| SSE event serialization (`event`/`data`/`id`/`retry`)            | FULLY_FUNCTIONAL | `WriteEvent` in `event.go`; `event_test.go`                                                  |
| Content-type + event-name constants (`ContentType`, `EventConnected`, `EventHeartbeat`) | FULLY_FUNCTIONAL | `constants.go`; used by `example/server.go`                                                  |
| Multi-line `data:` splitting (LF, CRLF, and lone CR per spec)    | FULLY_FUNCTIONAL | `splitLines` in `event.go`; `TestWriteEvent_CRLFInData`, `TestWriteEvent_LoneCarriageReturn` |
| Allocation-minimized writer (byte appends, no `fmt` on hot path) | FULLY_FUNCTIONAL | `WriteEvent` in `event.go`                                                                   |
| Heartbeat comment frames                                         | FULLY_FUNCTIONAL | `WriteHeartbeat` in `event.go`; `TestWriteHeartbeat`                                         |
| Retry directive (`uint`, rejects negative at compile time)       | FULLY_FUNCTIONAL | `WriteRetry` in `event.go`; `TestWriteRetry`, `TestWriteEvent_Retry`                         |
| `Event.String()` debug representation (omits empty fields)       | FULLY_FUNCTIONAL | `Event.String` in `event.go`; `TestEvent_String`                                             |

## Connection management (`Stream`)

| Feature                                                              | Status           | Evidence                                                                                                                    |
| -------------------------------------------------------------------- | ---------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Stream lifecycle: SSE headers + `200 OK` on creation                 | FULLY_FUNCTIONAL | `NewStream` in `stream.go`; `stream_test.go`                                                                                |
| `SetHeaders` (set SSE headers without writing status, for custom handlers) | FULLY_FUNCTIONAL | `SetHeaders` in `stream.go`; used internally by `NewStream`                                                                 |
| Send event + flush                                                   | FULLY_FUNCTIONAL | `Stream.Send` in `stream.go`; `TestStream_Send`                                                                             |
| Concurrent-safe writes (mutex-serialized `Send`/`Heartbeat`/`Close`) | FULLY_FUNCTIONAL | `Stream.mu` in `stream.go`; `TestStream_SendHeartbeatRaceSafety`, `TestStream_SendHeartbeatCloseRace` (race-tested)         |
| Heartbeat goroutine (proxy keep-alive)                               | FULLY_FUNCTIONAL | `Stream.Heartbeat` in `stream.go`; `TestStream_Heartbeat`                                                                   |
| `Last-Event-ID` header extraction (validated via `ParseEventID`)     | FULLY_FUNCTIONAL | `LastEventIDFromRequest` in `stream.go`; `TestLastEventIDFromRequest_MaliciousInputTreatedAsEmpty`                          |
| `OnDisconnect` callbacks (ordered)                                   | FULLY_FUNCTIONAL | `Stream.OnDisconnect` in `stream.go`; `TestStream_OnDisconnectMultipleInOrder`                                              |
| `SendData` convenience                                               | FULLY_FUNCTIONAL | `Stream.SendData` in `stream.go`; `TestStream_SendData`                                                                     |
| `SendJSON` convenience (marshal + send)                              | FULLY_FUNCTIONAL | `Stream.SendJSON` in `stream.go`; `TestStream_SendJSON`, `TestStream_SendJSON_MarshalError`, `TestStream_SendJSON_NilValue` |
| `Send` error on write failure (disconnected client)                  | FULLY_FUNCTIONAL | `Stream.Send` in `stream.go`; `TestStream_SendReturnsErrorOnWriteFailure`                                                   |
| Concurrent `Send`+`Close` race safety                                | FULLY_FUNCTIONAL | `TestStream_SendCloseRace`, `TestStream_SendHeartbeatCloseRace` (three-way race-tested)                                     |
| Request-context cancellation                                         | FULLY_FUNCTIONAL | `Stream.Context` in `stream.go`; `TestStream_ContextCancellation`                                                           |
| `io.Closer` interface compliance                                     | FULLY_FUNCTIONAL | `Stream.Close` in `stream.go`; `TestStream_DoubleCloseSafety`                                                               |

## Fan-out (`Broadcaster[T]` / `fanOut[T]`)

| Feature                                              | Status           | Evidence                                                                                                    |
| ---------------------------------------------------- | ---------------- | ----------------------------------------------------------------------------------------------------------- |
| Generic fan-out over any message type                | FULLY_FUNCTIONAL | `Broadcaster[T]` in `broadcaster.go`; `TestBroadcaster_GenericWithString`                                   |
| Subscribe / Unsubscribe                              | FULLY_FUNCTIONAL | `fanOut.Subscribe`, `fanOut.Unsubscribe` in `fanout.go`; `TestBroadcaster_Subscribe`                        |
| O(1) unsubscribe via channel pointer identity        | FULLY_FUNCTIONAL | `channelPtr` in `fanout.go`; `TestBroadcaster_BroadcastUnsubscribeRace`                                     |
| Non-blocking broadcast (drops to slow consumers)     | FULLY_FUNCTIONAL | `fanOut.Broadcast` in `fanout.go`; `TestBroadcaster_DropsOnFullBuffer`                                      |
| Batch broadcast (`BroadcastMany`, single lock pass)  | FULLY_FUNCTIONAL | `fanOut.BroadcastMany` in `fanout.go`; `TestBroadcaster_BroadcastMany`, `_PreservesOrder`, `_MixedSlowFast` |
| Broadcast after Close is safe (no panic)             | FULLY_FUNCTIONAL | `TestBroadcaster_BroadcastAfterClose`                                                                       |
| `OnSubscribe` / `OnUnsubscribe` hooks                | FULLY_FUNCTIONAL | `fanOut.OnSubscribe`, `fanOut.OnUnsubscribe` in `fanout.go`; `TestBroadcaster_OnSubscribeHook`              |
| Graceful `Close` (closes all channels)               | FULLY_FUNCTIONAL | `fanOut.Close` in `fanout.go`; `TestBroadcaster_Close`                                                      |
| Subscribe-after-close returns closed channel (no-op) | FULLY_FUNCTIONAL | `fanOut.Subscribe` in `fanout.go`; `TestBroadcaster_SubscribeAfterClose`                                    |
| Concurrent-safety (race-tested)                      | FULLY_FUNCTIONAL | `TestBroadcaster_ConcurrentSafety`, `TestBroadcaster_ConcurrentHookCount`                                   |

## Reconnection replay

| Feature                                                        | Status           | Evidence                                                                  |
| -------------------------------------------------------------- | ---------------- | ------------------------------------------------------------------------- |
| `EventStore` interface (consumer-implemented, takes `EventID`) | FULLY_FUNCTIONAL | `replay.go`                                                               |
| `Replay` function (replays missed events after a given ID)     | FULLY_FUNCTIONAL | `Replay` in `replay.go`; `TestReplay_AfterGivenID`, `TestReplay_NoLastID` |
| Write-failure error propagation                                | FULLY_FUNCTIONAL | `Replay` in `replay.go`; `TestReplay_WriteError`                          |
| Store-error propagation (`EventsAfter` returns error)          | FULLY_FUNCTIONAL | `Replay` in `replay.go`; `TestReplay_StoreError`                          |

## Type safety

| Feature                                                             | Status           | Evidence                                                                                     |
| ------------------------------------------------------------------- | ---------------- | -------------------------------------------------------------------------------------------- |
| Branded `EventID` (prevents cross-assignment with other string IDs) | FULLY_FUNCTIONAL | `event.go`; `TestEventID_IsZero`                                                             |
| `ParseEventID` (rejects `\n`/`\r` that corrupt the wire format)     | FULLY_FUNCTIONAL | `ParseEventID` in `event.go`; `TestParseEventID_RejectsNewlines`, `TestParseEventID_Unicode` |
| `MustParseEventID` (panicking variant for tests/constants)          | FULLY_FUNCTIONAL | `MustParseEventID` in `event.go`; `TestMustParseEventID_Panics`                              |

## Testing infrastructure

| Feature               | Status           | Evidence                                                                                                                                          |
| --------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Fuzz tests            | FULLY_FUNCTIONAL | `FuzzWriteEvent`, `FuzzParseEventID` in `fuzz_test.go`                                                                                            |
| Integration tests     | FULLY_FUNCTIONAL | `TestIntegration_DirectSendAndHeaders`, `TestIntegration_BroadcasterFanOut` in `integration_test.go`                                              |
| Race detector tests   | FULLY_FUNCTIONAL | `TestStream_SendHeartbeatRaceSafety`, `TestStream_SendCloseRace`, `TestStream_SendHeartbeatCloseRace`, `TestBroadcaster_BroadcastUnsubscribeRace` |
| Example tests (godoc) | FULLY_FUNCTIONAL | `ExampleWriteEvent`, `ExampleBroadcaster`, `ExampleParseEventID` in `example_test.go`                                                             |
| Benchmarks            | FULLY_FUNCTIONAL | `BenchmarkBroadcasterFanOut` (1–10k subs), `BenchmarkBroadcastManyVsLoop` in `broadcaster_test.go`                                                |
| CI pipeline           | FULLY_FUNCTIONAL | `.github/workflows/ci.yml`                                                                                                                        |

## Explicit non-goals

These are deliberately out of scope. They are documented here so contributors
do not mistake their absence for a gap.

- WebSocket support
- CQRS dispatch hooks
- Dashboard server, routes, or HTML templates
- Event bus integration
- Any opinion about payload format (strings, JSON, HTML fragments are all fine)

## Dependencies

Two same-author utility modules (the only `require` entries in `go.mod`):

- `github.com/larsartmann/go-branded-id` — phantom-type branded IDs (`EventID`)
- `github.com/larsartmann/go-error-family` — structured error wrapping

Requires Go 1.26+ with `GOEXPERIMENT=jsonv2` (transitive, via `go-branded-id`).
