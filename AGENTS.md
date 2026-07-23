# AGENTS.md — go-sse

Server-Sent Events transport library for Go. Single package (`sse`), flat layout. Wire format + connection lifecycle + fan-out + reconnection replay. Two small dependencies (`go-branded-id`, `go-error-family`); no domain opinions.

## Commands

No `flake.nix`, `Makefile`, or CI config. Use raw `go` tooling (from `CONTRIBUTING.md`):

```bash
GOWORK=off GOEXPERIMENT=jsonv2 go test ./... -race -count=1   # tests
GOWORK=off GOEXPERIMENT=jsonv2 go vet ./...                    # vet
GOWORK=off GOEXPERIMENT=jsonv2 golangci-lint run ./...         # lint
```

**`GOEXPERIMENT=jsonv2` is required** to build, transitively via `go-branded-id`. Without it, compilation fails. Always prefix build/test/vet commands with it.

**`GOWORK=off` is required in this environment.** A parent `/home/lars/projects/go.work` includes sibling projects (cqrs-htmx, etc.). One sibling (`cqrs-htmx`) has a stale checksum in its `go.sum` for `go-cqrs-lite/query/v4@v4.0.2`, which causes a `SECURITY ERROR` checksum mismatch when workspace mode resolves the combined module graph. `GOWORK=off` isolates go-sse to its own (valid) `go.mod`/`go.sum`. The `go.work` file is gitignored and does not exist in a fresh clone — external contributors do not need this flag.

## Dependencies

Only two, both `github.com/larsartmann/*`:

- `go-branded-id` — phantom-type branded IDs (`EventID = brandid.ID[eventBrand, string]`)
- `go-error-family` — structured error wrapping with severity categories

## Architecture

Four layers, each in its own file, composable independently:

| Layer             | File                           | Role                                                                |
| ----------------- | ------------------------------ | ------------------------------------------------------------------- |
| Wire format       | `event.go`                     | `Event`, `EventID`, `WriteEvent`, `WriteHeartbeat`, `WriteRetry`    |
| Single connection | `stream.go`                    | `Stream` — headers, mutex-guarded send, heartbeat, disconnect hooks |
| Fan-out           | `broadcaster.go` + `fanout.go` | `Broadcaster[T]` (public) embeds `fanOut[T]` (unexported hub)       |
| Reconnection      | `replay.go`                    | `EventStore` interface + `Replay` function                          |

**Data flow:** `Broadcaster.Broadcast(evt)` → non-blocking `select` send into each subscriber's buffered channel → handler's `select` loop reads channel → `stream.Send(evt)` → `WriteEvent` → `ResponseWriter.Write` + `Flush`.

### Broadcaster vs fanOut split

`Broadcaster[T]` is a thin public wrapper (`type Broadcaster[T any] struct{ *fanOut[T] }`). All real logic lives in the unexported `fanOut[T]` in `fanout.go`. This split exists so the fan-out hub is transport-agnostic and reusable (e.g. for non-SSE message fan-out). When adding broadcaster behavior, edit `fanout.go`, not `broadcaster.go`.

## Concurrency Invariants (critical)

1. **`Stream.mu` serializes ALL writes** to the underlying `ResponseWriter`. `Send`, `Heartbeat`, and `Close` all acquire it because `http.ResponseWriter` is not safe for concurrent use. Any new write path must hold this mutex.
2. **`fanOut` uses RWMutex with non-blocking sends.** `Broadcast` holds `RLock` during iteration and sends via `select { case ch <- msg: default: }` (drop). The read lock guarantees `Unsubscribe` cannot close a channel mid-send. Never change Broadcast to a blocking send under the read lock.
3. **`fanOut.Close()` sets `subscribers = nil` as the closed sentinel.** `Subscribe` checks for nil and returns an already-closed channel. Don't repurpose nil to mean "uninitialized."

## Gotchas

- **`NewStream` writes `200 OK` immediately** and sets headers — it is not lazy. Call it once per handler; do not write to `w` before `NewStream`.
- **`flusher` is an unexported interface** (`interface{ Flush() }`), not `http.Flusher`. `NewStream` does `w.(flusher)` which silently yields `nil` for non-flushing writers; `Send`/`Heartbeat` nil-check before flushing.
- **Broadcast is intentionally lossy.** A 64-deep buffer per subscriber (`defaultSubscriberBuffer`); overflow drops silently. Consumers needing guaranteed delivery must implement app-level ack/replay. Do not "fix" this into a blocking send — it would cause head-of-line blocking.
- **`MustParseEventID` panics** — it is for tests and constants only, never untrusted input. Use `ParseEventID` (rejects `\n`/`\r` that would corrupt the wire format) for request-header values.
- **Empty `EventID` is the zero/initial-connection value**, not an error. `Replay` with an empty ID replays everything (the store decides semantics).
- **`OnDisconnect` callbacks fire inside `Close()`**, after the mutex is released, in registration order.

## Conventions

- **Allocation-free hot path:** `WriteEvent` uses direct byte appends (`buf = append(buf, 'e','v','e','n','t',':',' ')`) rather than `fmt.Fprintf`. `WriteRetry` uses `fmt.Fprintf` only because it's not hot. Preserve the append style when extending `WriteEvent`.
- **`//nolint:wrapcheck`** marks functions where raw errors are intentionally returned unwrapped because the underlying write error is already actionable (`WriteHeartbeat`, `WriteRetry`). `WriteEvent` _does_ wrap via `errorfamily.Wrapf` with the `Transient` severity.
- **Errors use `go-error-family`** with stable codes (`"sse.write_failed"`, `"sse.event_id_invalid"`, `"sse.replay_failed"`) and severity categories (`Transient`, `Rejection`). Match this pattern for new errors.
- **Go 1.26 idioms in use:** integer range loops (`for i := range len(s)`, `for range 65`).
- **Tests:** external package `sse_test`, every test starts with `t.Parallel()`, `httptest` for stream tests, `bytes.Buffer` for serialization, `errorWriter`/`errorResponseWriter` fakes for failure paths. Race tests exist (`TestBroadcaster_BroadcastUnsubscribeRace`).
- **`splitLines`** (event.go) handles the SSE multi-line `data:` requirement and CRLF stripping; the no-newline fast path returns a single-element slice without allocating a backing array.

## What This Library Is NOT

No CQRS, no dashboard/routes/templates, no WebSockets, no event bus, no payload-format opinion. Adding any of these breaks the library's scope. Consumers build domain layers on top.
