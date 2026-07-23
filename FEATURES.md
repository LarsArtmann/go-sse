# Features

Honest inventory of what `go-sse` does and its real status. Status reflects the
code in this repository, verified against the test suite (`*_test.go`).

## Status vocabulary

| Status               | Meaning                                                                    |
| -------------------- | -------------------------------------------------------------------------- |
| FULLY_FUNCTIONAL     | Code present AND working (tests pass or exercised).                        |
| PARTIALLY_FUNCTIONAL | Ships but has known gaps, edge-case bugs, or missing pieces.               |
| BROKEN               | Code exists but does not work / is disabled / fails.                       |
| NOT_INCLUDED         | Deliberately out of scope — no code, by design (see "Explicit non-goals"). |

## Wire format & serialization

| Feature                                                          | Status           | Evidence                                                |
| ---------------------------------------------------------------- | ---------------- | ------------------------------------------------------- |
| SSE event serialization (`event`/`data`/`id`/`retry`)            | FULLY_FUNCTIONAL | `event.go:96` (`WriteEvent`); `event_test.go`           |
| Multi-line `data:` splitting (per spec)                          | FULLY_FUNCTIONAL | `event.go:159` (`splitLines`); `event_test.go`          |
| CRLF handling in `data`                                          | FULLY_FUNCTIONAL | `event.go:175`; `TestWriteEvent_CRLFInData`             |
| Allocation-minimized writer (byte appends, no `fmt` on hot path) | FULLY_FUNCTIONAL | `event.go:96`                                           |
| Heartbeat comment frames                                         | FULLY_FUNCTIONAL | `event.go:140` (`WriteHeartbeat`); `TestWriteHeartbeat` |
| Retry directive                                                  | FULLY_FUNCTIONAL | `event.go:149` (`WriteRetry`); `TestWriteEvent_Retry`   |

## Connection management (`Stream`)

| Feature                                                              | Status           | Evidence                                                           |
| -------------------------------------------------------------------- | ---------------- | ------------------------------------------------------------------ |
| Stream lifecycle: SSE headers + `200 OK` on creation                 | FULLY_FUNCTIONAL | `stream.go:67` (`NewStream`); `stream_test.go`                     |
| Send event + flush                                                   | FULLY_FUNCTIONAL | `stream.go:89` (`Send`); `TestStream_Send`                         |
| Concurrent-safe writes (mutex-serialized `Send`/`Heartbeat`/`Close`) | FULLY_FUNCTIONAL | `stream.go:46`; `TestStream_SendHeartbeatRaceSafety` (race-tested) |
| Heartbeat goroutine (proxy keep-alive)                               | FULLY_FUNCTIONAL | `stream.go:152`; `TestStream_Heartbeat`                            |
| `Last-Event-ID` header extraction                                    | FULLY_FUNCTIONAL | `stream.go:137`; `TestStream_LastEventID`                          |
| `OnDisconnect` callbacks (ordered)                                   | FULLY_FUNCTIONAL | `stream.go:180`; `TestStream_OnDisconnectMultipleInOrder`          |
| `SendHTML` convenience                                               | FULLY_FUNCTIONAL | `stream.go:106`; `TestStream_SendHTML`                             |
| Request-context cancellation                                         | FULLY_FUNCTIONAL | `stream.go:113`; `TestStream_ContextCancellation`                  |

## Fan-out (`Broadcaster[T]` / `fanOut[T]`)

| Feature                                              | Status           | Evidence                                                                   |
| ---------------------------------------------------- | ---------------- | -------------------------------------------------------------------------- |
| Generic fan-out over any message type                | FULLY_FUNCTIONAL | `broadcaster.go:23`; `TestBroadcaster_GenericWithString`                   |
| Subscribe / Unsubscribe                              | FULLY_FUNCTIONAL | `fanout.go:39`, `fanout.go:63`; `TestBroadcaster_Subscribe`                |
| O(1) unsubscribe via channel pointer identity        | FULLY_FUNCTIONAL | `fanout.go:149` (`channelPtr`); `TestBroadcaster_BroadcastUnsubscribeRace` |
| Non-blocking broadcast (drops to slow consumers)     | FULLY_FUNCTIONAL | `fanout.go:89`; `TestBroadcaster_DropsOnFullBuffer`                        |
| `OnSubscribe` / `OnUnsubscribe` hooks                | FULLY_FUNCTIONAL | `fanout.go:131`, `fanout.go:141`; `TestBroadcaster_OnSubscribeHook`        |
| Graceful `Close` (closes all channels)               | FULLY_FUNCTIONAL | `fanout.go:116`; `TestBroadcaster_Close`                                   |
| Subscribe-after-close returns closed channel (no-op) | FULLY_FUNCTIONAL | `fanout.go:43`; `TestBroadcaster_SubscribeAfterClose`                      |
| Concurrent-safety (race-tested)                      | FULLY_FUNCTIONAL | `TestBroadcaster_ConcurrentSafety`, `TestBroadcaster_ConcurrentHookCount`  |

## Reconnection replay

| Feature                                                    | Status           | Evidence                                                         |
| ---------------------------------------------------------- | ---------------- | ---------------------------------------------------------------- |
| `EventStore` interface (consumer-implemented)              | FULLY_FUNCTIONAL | `replay.go:9`                                                    |
| `Replay` function (replays missed events after a given ID) | FULLY_FUNCTIONAL | `replay.go:21`; `TestReplay_AfterGivenID`, `TestReplay_NoLastID` |
| Write-failure error propagation                            | FULLY_FUNCTIONAL | `replay.go:26`; `TestReplay_WriteError`                          |

## Type safety

| Feature                                                             | Status           | Evidence                                          |
| ------------------------------------------------------------------- | ---------------- | ------------------------------------------------- |
| Branded `EventID` (prevents cross-assignment with other string IDs) | FULLY_FUNCTIONAL | `event.go:26`; `TestEventID_IsZero`               |
| `ParseEventID` (rejects `\n`/`\r` that corrupt the wire format)     | FULLY_FUNCTIONAL | `event.go:44`; `TestParseEventID_RejectsNewlines` |
| `MustParseEventID` (panicking variant for tests/constants)          | FULLY_FUNCTIONAL | `event.go:55`; `TestMustParseEventID_Panics`      |

## Explicit non-goals (NOT_INCLUDED, by design)

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
