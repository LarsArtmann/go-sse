# AGENTS.md — go-sse

Server-Sent Events transport library for Go. Single package (`sse`), flat layout. Wire format + connection lifecycle + fan-out + reconnection replay. Two small dependencies (`go-branded-id`, `go-error-family`); no domain opinions.

## Commands

`flake.nix` provides hermetic build/test/lint. The devShell sets `GOWORK=off` and `GOEXPERIMENT=jsonv2` automatically:

```bash
nix run .#test-race          # tests with race detector
nix run .#vet                # go vet
nix run .#lint               # golangci-lint
nix run .#coverage           # test + coverage report
nix flake check              # full hermetic check (compile + test)
nix develop                  # enter dev shell (Go 1.26, golangci-lint, gopls, govulncheck, templ)
```

Formatting via treefmt (`nix fmt`): gofumpt, goimports, golines, nixfmt.
Generated `*_templ.go` files are excluded from treefmt and golangci-lint.

Raw `go` tooling (for environments without Nix):

```bash
GOWORK=off GOEXPERIMENT=jsonv2 go test ./... -race -count=1   # tests
GOWORK=off GOEXPERIMENT=jsonv2 go vet ./...                    # vet
GOWORK=off GOEXPERIMENT=jsonv2 golangci-lint run ./...         # lint
templ generate                                                 # regenerate *_templ.go from .templ files
```

**`GOEXPERIMENT=jsonv2` is required** to build, transitively via `go-branded-id`. Without it, compilation fails. Always prefix build/test/vet commands with it.

**`GOWORK=off` is required in this environment.** A parent `/home/lars/projects/go.work` includes sibling projects (cqrs-htmx, etc.). One sibling (`cqrs-htmx`) has a stale checksum in its `go.sum` for `go-cqrs-lite/query/v4@v4.0.2`, which causes a `SECURITY ERROR` checksum mismatch when workspace mode resolves the combined module graph. `GOWORK=off` isolates go-sse to its own (valid) `go.mod`/`go.sum`. The `go.work` file is gitignored and does not exist in a fresh clone — external contributors do not need this flag. (The `flake.nix` devShell sets `GOWORK=off` automatically.)

**`buildflow` needs direnv (`.envrc`), not just the devShell.** `buildflow` is a system binary that inherits the parent shell's environment — it does **not** read the `flake.nix` devShell and does **not** support env configuration in `.buildflow.yml`. The devShell's `GOEXPERIMENT`/`GOWORK` `mkShell` attributes only apply inside `nix develop`; they do not reach tools launched from a normal shell. The project's `.envrc` (`use flake` + explicit `export GOEXPERIMENT=jsonv2` / `export GOWORK=off`) is what propagates them to `buildflow`, `gopls`, and direct `go` invocations via direnv. **Symptom of a missing or un-`direnv allow`ed `.envrc`:** `buildflow`'s `go-fix`, `test-race`, and `govalid-generate` fail with `imports encoding/json/v2: build constraints exclude all Go files`. `.envrc` is gitignored (buildflow-managed); each contributor creates it locally — copy the pattern from `dnsblockd` if absent.

## Dependencies

Two library dependencies, both `github.com/larsartmann/*`:

- `go-branded-id` — phantom-type branded IDs (`EventID = brandid.ID[eventBrand, string]`)
- `go-error-family` — structured error wrapping with severity categories

One example-only dependency:

- `a-h/templ` — type-safe HTML templates for the browser examples (`example/datastar/` and `example/htmx/`). The library itself does not import templ; it is only used by the example `main` packages. The generated `*_templ.go` files are checked into git, so consumers of the library never need the `templ` CLI.

## Architecture

Four layers, each in its own file, composable independently:

| Layer             | File                           | Role                                                                                                                                                                                                                           |
| ----------------- | ------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Wire format       | `event.go`                     | `Event`, `EventID`, `WriteEvent`, `WriteHeartbeat`, `WriteRetry`, `KeyedLines`, `WriteKeyedLines`                                                                                                                              |
| Single connection | `stream.go`                    | `Stream` — headers, mutex-guarded send, heartbeat, disconnect hooks, `SendLines`, `SendKeyed`                                                                                                                                  |
| Fan-out           | `broadcaster.go` + `fanout.go` | `Broadcaster[T]` (public) embeds `fanOut[T]` (unexported hub). `SubscribeFilter` for predicate-based filtering. `Shutdown(ctx)` + `Health()` for graceful lifecycle. `WithBufferSize[T]` for configurable subscriber capacity. |
| Reconnection      | `replay.go`                    | `EventStore` interface + `Replay` function. `FilteredEventStore` + `ReplayFiltered` for predicate-aware replay.                                                                                                                |

**Data flow:** `Broadcaster.Broadcast(evt)` → non-blocking `select` send into each subscriber's buffered channel → handler's `select` loop reads channel → `stream.Send(evt)` → `WriteEvent` → `ResponseWriter.Write` + `Flush`.

### Broadcaster vs fanOut split

`Broadcaster[T]` is a thin public wrapper (`type Broadcaster[T any] struct{ *fanOut[T] }`). All real logic lives in the unexported `fanOut[T]` in `fanout.go`. This split exists so the fan-out hub is transport-agnostic and reusable (e.g. for non-SSE message fan-out). When adding broadcaster behavior, edit `fanout.go`, not `broadcaster.go`.

## Concurrency Invariants (critical)

1. **`Stream.mu` serializes ALL writes** to the underlying `ResponseWriter`. `Send`, `Heartbeat`, and `Close` all acquire it because `http.ResponseWriter` is not safe for concurrent use. Any new write path must hold this mutex.
2. **`fanOut` uses RWMutex with non-blocking sends.** `Broadcast` holds `RLock` during iteration and sends via `select { case ch <- msg: default: }` (drop). The read lock guarantees `Unsubscribe` cannot close a channel mid-send. Never change Broadcast to a blocking send under the read lock. **SubscribeFilter predicates are called under this same read lock** — they must be pure, fast, and non-blocking.
3. **`fanOut.Close()` sets `subscribers = nil` as the closed sentinel.** `Subscribe` checks for nil and returns an already-closed channel. Don't repurpose nil to mean "uninitialized."
4. **`fanOut.draining` is the "shutdown in progress" sentinel.** Distinct from the closed sentinel so `Subscribe` can keep returning closed channels during a drain while `Close` is not yet meaningful. Always set draining under the write lock and check it under the read or write lock.

## Lifecycle API

`Broadcaster` now exposes a graceful-shutdown story that mirrors `http.Server`:

- `Close()` — instant shutdown; closes all subscriber channels immediately. Use for hard shutdown.
- `Shutdown(ctx)` — graceful drain; marks the broadcaster as draining (rejects new `Subscribe` calls), waits for every active subscriber's buffer to be empty (consumers caught up), then closes the channels. Returns the context's error (wrapped via `errorfamily` with code `sse.shutdown_drain_deadline_exceeded`) if the deadline fires before the drain completes. The caller can retry with a fresh context or fall back to `Close`.
- `Health() BroadcasterHealth` — value-type snapshot of `Closed`, `Draining`, `SubscriberCount`, and `BufferSize`. Cheap (read-lock + struct copy). Suitable for k8s liveness/readiness probes.
- `NewBroadcaster[T](opts ...Option[T])` — accepts functional options at construction. The only option today is `WithBufferSize[T](size int)`. Buffer size is read once and not changed later; non-positive values are silently ignored (default kept).

## Gotchas

- **`NewStream` writes `200 OK` immediately** and sets headers — it is not lazy. Call it once per handler; do not write to `w` before `NewStream`.
- **`flusher` is an unexported interface** (`interface{ Flush() }`), not `http.Flusher`. `NewStream` does `w.(flusher)` which silently yields `nil` for non-flushing writers; `Send`/`Heartbeat` nil-check before flushing.
- **Broadcast is intentionally lossy.** A 64-deep buffer per subscriber (`defaultSubscriberBuffer`); overflow drops silently. Consumers needing guaranteed delivery must implement app-level ack/replay. Do not "fix" this into a blocking send — it would cause head-of-line blocking.
- **`MustParseEventID` panics** — it is for tests and constants only, never untrusted input. Use `ParseEventID` (rejects `\n`/`\r` that would corrupt the wire format) for request-header values.
- **Empty `EventID` is the zero/initial-connection value**, not an error. `Replay` with an empty ID replays everything (the store decides semantics).
- **`OnDisconnect` callbacks fire inside `Close()`**, after the mutex is released, in registration order.
- **`SubscribeFilter` predicates run under the fanOut read lock.** The predicate is called once per subscriber per broadcast inside `sendAllLocked`. It must be pure (no side effects), fast (no I/O, no blocking), and non-blocking. Nil predicate means "all events" (identical to `Subscribe`).
- **Predicate panics are recovered and treated as non-matches.** `safePredCall` (in `fanout.go`) wraps the predicate call in `defer/recover`; a panicking predicate causes the event to be skipped for that subscriber only — one broken predicate cannot crash the broadcaster or replay loop. This applies to both `SubscribeFilter`'s fan-out path and `ReplayFiltered`'s fallback path. The `EventsAfterFiltered` path on a `FilteredEventStore` is the store's responsibility; document the store's panic contract there.
- **`ReplayFiltered` type-asserts to `FilteredEventStore`.** If the store implements the interface, the predicate is pushed into the store query (efficient). Otherwise it falls back to `EventsAfter` + in-memory post-filter (correct but budget-inefficient). Nil pred delegates to `Replay`.
- **Internal `subscriber[T]` struct** wraps `chan T` + optional `func(T) bool` predicate. The subscriber map is `map[uintptr]*subscriber[T]`, keyed by channel pointer identity (same as before). One allocation per Subscribe call (negligible — once per connection, not per event).
- **`Shutdown` polls at `drainPollInterval` (1ms).** The drain loop checks each subscriber's channel length (`len(sub.ch) > 0`) and re-checks on each tick. A subscriber that is genuinely slow will keep Shutdown waiting indefinitely until the context fires. The 1ms granularity is short enough that an idle consumer registers promptly, long enough to avoid burning CPU when there are many slow consumers.
- **`Shutdown`'s deadline error preserves `errors.Is(err, context.Canceled)` / `context.DeadlineExceeded`.** The error is wrapped with `errorfamily.Wrapf` but the underlying ctx.Err() is the Unwrap target, so existing context-cancellation checks keep working.
- **`Shutdown` is safe to call from multiple goroutines but only the first call's drain takes effect.** Subsequent calls return nil once the hub is already closed.
- **`WithBufferSize` non-positive values are silently ignored.** The default (`defaultSubscriberBuffer = 64`) is kept. This means `WithBufferSize(0)` and `WithBufferSize(-1)` are no-ops; pass a positive integer.

## Conventions

- **Allocation-free hot path:** `WriteEvent` uses direct byte appends (`buf = append(buf, 'e','v','e','n','t',':',' ')`) rather than `fmt.Fprintf`. `WriteRetry` uses `fmt.Fprintf` only because it's not hot. Preserve the append style when extending `WriteEvent`.
- **`//nolint:wrapcheck`** marks functions where raw errors are intentionally returned unwrapped because the underlying write error is already actionable (`WriteHeartbeat`, `WriteRetry`). `WriteEvent` _does_ wrap via `errorfamily.Wrapf` with the `Transient` severity.
- **Errors use `go-error-family`** with stable codes (`"sse.write_failed"`, `"sse.event_id_invalid"`, `"sse.replay_failed"`) and severity categories (`Transient`, `Rejection`). Match this pattern for new errors.
- **Go 1.26 idioms in use:** integer range loops (`for i := range len(s)`, `for range 65`).
- **Tests:** external package `sse_test`, every test starts with `t.Parallel()`, `httptest` for stream tests, `bytes.Buffer` for serialization, `errorWriter`/`errorResponseWriter` fakes for failure paths. Race tests exist (`TestBroadcaster_BroadcastUnsubscribeRace`).
- **`splitLines`** (event.go) handles the SSE multi-line `data:` requirement and CRLF stripping; the no-newline fast path returns a single-element slice without allocating a backing array.
- **`KeyedLines`** (event.go) prefixes each line of a multi-line value with `key `, producing the newline-joined string for `Event.Data`. This is the building block for keyed-data-line protocols like DataStar (`data: elements <div>` / `data: elements </div>`). It delegates to `splitLines` for line splitting. Returns `""` for empty input (no data line emitted).
- **`Stream.SendLines`** (stream.go) is a convenience wrapper around `Send` that joins variadic string arguments with `\n` into `Event.Data`. Composes with `KeyedLines` for the DataStar pattern: each `KeyedLines` result (which may contain embedded `\n`) is one argument, and `splitLines` in `WriteEvent` handles the final split.
- **`WriteKeyedLines`** (event.go) is the wire-only single-key convenience: `WriteEvent(w, Event{Event: eventType, Data: KeyedLines(key, value)})`. For consumers that use `WriteEvent` directly without a `Stream`.
- **`Stream.SendKeyed`** (stream.go) is the stream-level single-key convenience: `Send(Event{Event: eventName, Data: KeyedLines(key, value)})`. For the common single-key DataStar pattern (e.g., `patch-signals`).

## Examples (`example/`)

Three runnable examples, each an independent `package main`:

- `example/server.go` — raw wire-format demo (curl-driven broadcast/fan-out), port `:8080`.
- `example/datastar/` — DataStar reactive-signal + DOM-patch UI, port `:8765`.
- `example/htmx/` — HTMX HTML-fragment-swap UI, port `:8766`.

**`example/README.md` compares the DataStar vs HTMX approaches in detail** (mechanism, payload size, granularity, bundle size, trade-offs). The two browser examples render the _same_ progress-bar demo through different mechanisms:

- **DataStar** uses go-sse's keyed-line helpers (`KeyedLines`/`SendKeyed`/`SendLines`) to patch reactive signals (`datastar-patch-signals`) and DOM elements (`datastar-patch-elements`).
- **HTMX** uses plain `stream.Send(Event{...})` to stream HTML fragments; the HTMX SSE extension (`sse-swap="progress"` + `hx-swap="innerHTML"`) swaps them into the DOM. No HTMX-specific helpers are needed — HTMX speaks vanilla SSE.

**Shared structure (both browser examples):**

- `index.templ` — type-safe HTML template (templ). Run `templ generate` after editing.
- `index_templ.go` — generated code (checked into git, excluded from treefmt and golangci-lint via `*_templ.go` path patterns).
- `static/styles.css` — CSS with dark/light theme support via `prefers-color-scheme`.
- `static/*.js` — client JS bundles, embedded via `go:embed` (no CDN).
- `main.go` — HTTP server: renders templ for `GET /`, serves embedded static files for `GET /static/`, SSE events for `GET /events`.

**Running:** `go run ./example/datastar/` or `go run ./example/htmx/` (run the package, not a single file, because the page render lives in the generated `index_templ.go`).

**Vendored client bundles:**

- DataStar: `static/datastar.js` (DataStar v1.0.2).
- HTMX: `static/htmx.min.js` (htmx 2.0.4) + `static/sse.min.js` (htmx-ext-sse 2.2.4). HTMX needs the separate SSE extension for `sse-connect`/`sse-swap`.

**Editing the template:** After changing `index.templ`, run `templ generate` to regenerate `index_templ.go`. The `templ` CLI is in the devShell.

**HTMX restart mechanism:** HTMX SSE connections are declared in markup (`sse-connect="/events"`). The Restart button fetches a fresh `#sse-container` fragment (`GET /sse-container`) and swaps it in (`outerHTML`), which tears down the old `EventSource` and opens a new one — no JavaScript. More ceremony than DataStar's one-attribute `@get('/events')`, but idiomatic HTMX.

## What This Library Is NOT

No CQRS, no dashboard/routes/templates, no WebSockets, no event bus, no payload-format opinion. Adding any of these breaks the library's scope. Consumers build domain layers on top.
