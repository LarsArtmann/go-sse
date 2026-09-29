# AGENTS.md — go-sse

Server-Sent Events transport library for Go. Single package (`sse`), flat layout, plus two nested modules: `sseparse/` (zero-dependency parser + assertions + WPT corpus-as-data) and `ssetest/` (HTTP collection helpers layered on sseparse via re-exports). Wire format + connection lifecycle + fan-out + reconnection replay. Two small root dependencies (`go-branded-id`, `go-error-family`); no domain opinions.

## Commands

`flake.nix` provides hermetic build/test/lint. The devShell sets `GOWORK=off` and `GOEXPERIMENT=jsonv2` automatically:

```bash
nix run .#test-race          # tests with race detector (root + ssetest + sseparse)
nix run .#vet                # go vet (root + ssetest + sseparse)
nix run .#lint               # golangci-lint (root + ssetest + sseparse)
nix run .#coverage           # test + coverage report (root + ssetest + sseparse)
nix run .#coverage-gate      # fail under thresholds (library 90%, ssetest 95%, sseparse 95%)
scripts/verify.sh            # one-command pre-push gate (fmt+vet+lint+test+flake check; --fast skips flake)
nix flake check              # full hermetic check (compile + test, root + ssetest + sseparse)
nix develop                  # enter dev shell (Go 1.27, golangci-lint, gopls, govulncheck, templ)
```

Formatting via treefmt (`nix fmt`): gofumpt, goimports, golines, nixfmt.
Generated `*_templ.go` files are excluded from treefmt and golangci-lint.

Raw `go` tooling (for environments without Nix):

```bash
GOWORK=off GOEXPERIMENT=jsonv2 go test ./... -race -count=1   # tests
GOWORK=off GOEXPERIMENT=jsonv2 go vet ./...                    # vet
GOWORK=off GOEXPERIMENT=jsonv2 golangci-lint run ./...         # lint
(cd ssetest && GOWORK=off GOEXPERIMENT=jsonv2 go test ./... -race -count=1)  # ssetest module
(cd sseparse && GOWORK=off GOEXPERIMENT=jsonv2 go test ./... -race -count=1) # sseparse module
templ generate                                                 # regenerate *_templ.go from .templ files
```

`ssetest/` and `sseparse/` are their own Go modules (`github.com/larsartmann/go-sse/ssetest` / `.../sseparse`). Neither carries `replace` directives since the 2026-09-29 release: ssetest consumes `go-sse` v0.6.0 and `sseparse` v0.1.0 as tagged modules, so a local, untagged sseparse change is only visible to ssetest tests after a temporary local `replace github.com/larsartmann/go-sse/sseparse => ../sseparse` (revert before committing) or a fresh sseparse tag; sseparse has ZERO third-party requires (test-guarded by `sseparse/go_mod_boundary_test.go` — the entire reason it exists, per the 2026-09-29 go-daemon feedback). Always `cd` into each module to build/test/lint it; `go test ./ssetest/...` from the root fails (module boundary). golangci-lint discovers the root `.golangci.yml` from inside both nested modules automatically (parent-directory search), so no separate config exists.

**`GOEXPERIMENT=jsonv2` is no longer required since the Go 1.27 toolchain** (verified 2026-09-19: both modules build and test with the variable unset — `encoding/json/v2` became a `go1.27`-gated stable API). The devShell, flake apps, CI, and scripts still export it as a harmless belt-and-braces default; removing it everywhere is future cleanup, not a correctness need. Two related traps: under the 1.26 toolchain it WAS required (transitively via `go-branded-id`), and — the trap that reddened master on 2026-09-18 — a module whose `go.mod` says `go 1.26.x` while using json/v2 does NOT compile under a 1.27 toolchain: bump root `go.mod`, `ssetest/go.mod`, and `sseparse/go.mod` TOGETHER to the same version (all three say `go 1.27` since the 2026-09-29 normalization — nothing needs more now that json/v2 is stable under 1.27; do not hand-raise a directive (correction 2026-09-29: plain `go mod tidy` does NOT revert one — tested; the historical reverts came from a normalize step — and `scripts/verify.sh` now fails the gate when the three modules' directives diverge, so a raised directive is caught, not silently flipflopped), or CI fails inside a nested module with "json.Unmarshal requires go1.27 or later (file is go1.26)".

**`GOWORK=off` is required in this environment.** A parent `/home/lars/projects/go.work` includes sibling projects (cqrs-htmx, etc.). One sibling (`cqrs-htmx`) has a stale checksum in its `go.sum` for `go-cqrs-lite/query/v4@v4.0.2`, which causes a `SECURITY ERROR` checksum mismatch when workspace mode resolves the combined module graph. `GOWORK=off` isolates go-sse to its own (valid) `go.mod`/`go.sum`. The `go.work` file is gitignored and does not exist in a fresh clone — external contributors do not need this flag. (The `flake.nix` devShell sets `GOWORK=off` automatically.)

**`buildflow` needs direnv (`.envrc`), not just the devShell.** `buildflow` is a system binary that inherits the parent shell's environment — it does **not** read the `flake.nix` devShell and does **not** support env configuration in `.buildflow.yml`. The devShell's `GOEXPERIMENT`/`GOWORK` `mkShell` attributes only apply inside `nix develop`; they do not reach tools launched from a normal shell. The project's `.envrc` (`use flake` + explicit `export GOEXPERIMENT=jsonv2` / `export GOWORK=off`) is what propagates them to `buildflow`, `gopls`, and direct `go` invocations via direnv. **Symptom of a missing or un-`direnv allow`ed `.envrc`:** `buildflow`'s `go-fix`, `test-race`, and `govalid-generate` fail with `imports encoding/json/v2: build constraints exclude all Go files`. `.envrc` is gitignored (buildflow-managed); each contributor creates it locally — copy the pattern from `dnsblockd` if absent.

## Dependencies

Two library dependencies, both `github.com/larsartmann/*`:

- `go-branded-id` — phantom-type branded IDs (`EventID = brandid.ID[eventBrand, string]`)
- `go-error-family` — structured error wrapping with severity categories

One example-only dependency:

- `a-h/templ` — type-safe HTML templates for the browser examples (`example/datastar/` and `example/htmx/`). The library itself does not import templ; it is only used by the example `main` packages. The generated `*_templ.go` files are checked into git, so consumers of the library never need the `templ` CLI.

## Architecture

Four layers, each in its own file, composable independently, plus a separate test-helpers module:

| Layer             | File                                        | Role                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| ----------------- | ------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Wire format       | `event.go`                                  | `Event`, `EventID`, `WriteEvent`, `WriteHeartbeat`, `WriteRetry`, `KeyedLines`, `WriteKeyedLines`                                                                                                                                                                                                                                                                                                                                                                                               |
| Single connection | `stream.go`                                 | `Stream` — headers, mutex-guarded send, heartbeat, disconnect hooks, `SendLines`, `SendKeyed`                                                                                                                                                                                                                                                                                                                                                                                                   |
| Fan-out           | `broadcaster.go` + `fanout.go`              | `Broadcaster[T]` (public) embeds `fanOut[T]` (unexported hub). `SubscribeFilter` for predicate-based filtering. `Shutdown(ctx)` + `Health()` for graceful lifecycle. `WithBufferSize[T]` for configurable subscriber capacity. `WithOnDrop[T]` + `OnDrop` for drop observability.                                                                                                                                                                                                               |
| Reconnection      | `replay.go`                                 | `EventStore` interface + `Replay` function. `FilteredEventStore` + `ReplayFiltered` for predicate-aware replay.                                                                                                                                                                                                                                                                                                                                                                                 |
| Test helpers      | `sseparse/` + `ssetest/` (separate modules) | `sseparse`: ZERO-dependency parser core — `ReadEvents`/`ReadNEvents`/`StreamReader` (WHATWG § 9.2.6-conformant), `Require*` assertions, and the WPT corpus exported as data (`Corpus()`/`MustCorpus`, single source of truth `sseparse/testdata/wpt_format_corpus.json`). `ssetest`: HTTP `Collect*` helpers + request options, re-exporting the whole sseparse API via `compat.go` (delegation pinned by `compat_test.go`). go-datastar's `datastartest` delegates its SSE parsing to ssetest. |

**Data flow:** `Broadcaster.Broadcast(evt)` → non-blocking `select` send into each subscriber's buffered channel → handler's `select` loop reads channel → `stream.Send(evt)` → `WriteEvent` → `ResponseWriter.Write` + `Flush`.

### Broadcaster vs fanOut split

`Broadcaster[T]` is a thin public wrapper (`type Broadcaster[T any] struct{ *fanOut[T] }`). All real logic lives in the unexported `fanOut[T]` in `fanout.go`. This split exists so the fan-out hub is transport-agnostic and reusable (e.g. for non-SSE message fan-out). When adding broadcaster behavior, edit `fanout.go`, not `broadcaster.go`.

## Concurrency Invariants (critical)

1. **`Stream.mu` serializes ALL writes** to the underlying `ResponseWriter`. `Send`, `Heartbeat`, and `Close` all acquire it because `http.ResponseWriter` is not safe for concurrent use. Any new write path must hold this mutex. `Close` also sets the `closed` flag under the mutex, and `Heartbeat` checks it before writing: a tick racing handler teardown must return instead of flushing a `ResponseWriter` the server has already finished — that exact race panicked inside `net/http` and reddened master on 2026-09-03 (`TestIntegration_HeartbeatDelivery`; pinned by `TestStream_HeartbeatStopsAfterClose`).
2. **`fanOut` uses RWMutex with non-blocking sends.** `Broadcast` holds `RLock` during iteration and sends via `select { case ch <- msg: default: }` (drop). The read lock guarantees `Unsubscribe` cannot close a channel mid-send. Never change Broadcast to a blocking send under the read lock. **SubscribeFilter predicates are called under this same read lock** — they must be pure, fast, and non-blocking.
3. **`fanOut.Close()` sets `subscribers = nil` as the closed sentinel.** `Subscribe` checks for nil and returns an already-closed channel. Don't repurpose nil to mean "uninitialized."
4. **`fanOut.draining` is the "shutdown in progress" sentinel.** Distinct from the closed sentinel so `Subscribe` can keep returning closed channels during a drain while `Close` is not yet meaningful. Always set draining under the write lock and check it under the read or write lock.
5. **`fanOut.onDrop` runs inside the read lock during `sendAllLocked`.** The callback fires on the broadcasting goroutine for each subscriber whose buffer is full. The `OnDrop` setter takes the write lock, so the field is stable mid-iteration — but the callback itself must be fast and non-blocking (same contract as `SubscribeFilter` predicates). Nil means no callback (nil-guarded in the drop branch). Since 2026-08-29 the invocation goes through `safeDropCall`: a panicking `onDrop` is recovered and the broadcast continues, symmetric with the predicate contract. Re-entrant broadcaster calls from the callback still deadlock (read lock vs the setters' write lock) — the `WithOnDrop`/`OnDrop` doc comments state this.

## Lifecycle API

`Broadcaster` now exposes a graceful-shutdown story that mirrors `http.Server`:

- `Close()` — instant shutdown; closes all subscriber channels immediately. Use for hard shutdown.
- `Shutdown(ctx)` — graceful drain; marks the broadcaster as draining (rejects new `Subscribe` calls), waits for every active subscriber's buffer to be empty (consumers caught up), then closes the channels. Returns the context's error (wrapped via `errorfamily` with code `sse.shutdown_drain_deadline_exceeded`) if the deadline fires before the drain completes. The caller can retry with a fresh context or fall back to `Close`.
- `Health() BroadcasterHealth` — value-type snapshot of `Closed`, `Draining`, `SubscriberCount`, and `BufferSize`. Cheap (read-lock + struct copy). Suitable for k8s liveness/readiness probes.
- `NewBroadcaster[T](opts ...Option[T])` — accepts functional options at construction. Options today are `WithBufferSize[T](size int)` and `WithOnDrop[T](fn func(T))`. Buffer size is read once and not changed later; non-positive values are silently ignored (default kept).
- `OnDrop(fn func(T))` — runtime setter for the drop callback, equivalent to the `WithOnDrop` constructor option but callable after construction (e.g. when the metrics registry is wired later). Pass nil to clear.

## Gotchas

- **`NewStream` writes `200 OK` immediately** and sets headers — it is not lazy. Call it once per handler; do not write to `w` before `NewStream`.
- **`flusher` is an unexported interface** (`interface{ Flush() }`), not `http.Flusher`. `NewStream` does `w.(flusher)` which silently yields `nil` for non-flushing writers; `Send`/`Heartbeat` nil-check before flushing.
- **Broadcast is intentionally lossy.** A 64-deep buffer per subscriber (`defaultSubscriberBuffer`); overflow drops silently. Consumers needing guaranteed delivery must implement app-level ack/replay. Do not "fix" this into a blocking send — it would cause head-of-line blocking.
- **`onDrop` fires per-subscriber, not per-message.** A single `Broadcast` to N subscribers with full buffers invokes `onDrop` N times with the same `msg`. A drop counter counts per-subscriber drops, not unique messages lost. This also applies to `BroadcastMany` — each message in the batch that overflows a buffer fires independently.
- **`MustParseEventID` panics** — it is for tests and constants only, never untrusted input. Use `ParseEventID` (rejects NUL, `\n`, and `\r`, the spec § 9.2.4 Last-Event-ID value space) for request-header values.
- **Empty `EventID` is the zero/initial-connection value**, not an error. `Replay` with an empty ID replays everything (the store decides semantics).
- **`OnDisconnect` callbacks fire inside `Close()`**, after the mutex is released, in registration order.
- **`SubscribeFilter` predicates run under the fanOut read lock.** The predicate is called once per subscriber per broadcast inside `sendAllLocked`. It must be pure (no side effects), fast (no I/O, no blocking), and non-blocking. Nil predicate means "all events" (identical to `Subscribe`).
- **Predicate panics are recovered and treated as non-matches.** `safePredCall` (in `fanout.go`) wraps the predicate call in `defer/recover`; a panicking predicate causes the event to be skipped for that subscriber only — one broken predicate cannot crash the broadcaster or replay loop. This applies to both `SubscribeFilter`'s fan-out path and `ReplayFiltered`'s fallback path. The `EventsAfterFiltered` path on a `FilteredEventStore` is the store's responsibility; document the store's panic contract there.
- **`ReplayFiltered` type-asserts to `FilteredEventStore`.** If the store implements the interface, the predicate is pushed into the store query (efficient). Otherwise it falls back to `EventsAfter` + in-memory post-filter (correct but budget-inefficient). Nil pred delegates to `Replay`.
- **Internal `subscriber[T]` struct** wraps `chan T` + optional `func(T) bool` predicate. The subscriber map is `map[uintptr]*subscriber[T]`, keyed by channel pointer identity (same as before). One allocation per Subscribe call (negligible — once per connection, not per event).
- **`Shutdown` polls at `drainPollInterval` (1ms) via `waitForDrain`.** The drain loop (extracted to `waitForDrain` helper) checks each subscriber's channel length (`len(subs[i].ch) > 0`) and re-checks on each `time.After` tick. Uses index-based loops and `time.After` (not a named ticker) to keep structural variables out of the error scope. A subscriber that is genuinely slow will keep Shutdown waiting indefinitely until the context fires. The 1ms granularity is short enough that an idle consumer registers promptly, long enough to avoid burning CPU when there are many slow consumers. The subscriber snapshot is built with `slices.Collect(maps.Values(f.subscribers))`.
- **`Shutdown`'s deadline error preserves `errors.Is(err, context.Canceled)` / `context.DeadlineExceeded`.** The error is wrapped with `errorfamily.Wrapf` but the underlying ctx.Err() is the Unwrap target, so existing context-cancellation checks keep working.
- **`Shutdown` is safe to call from multiple goroutines but only the first call's drain takes effect.** Subsequent calls return nil once the hub is already closed.
- **`WithBufferSize` non-positive values are silently ignored.** The default (`defaultSubscriberBuffer = 64`) is kept. This means `WithBufferSize(0)` and `WithBufferSize(-1)` are no-ops; pass a positive integer.
- **`sseparse` is the parser; `ssetest` re-exports it** (split 2026-09-29 from the go-daemon adoption feedback: consumers of the parser must not pay go-sse/go-branded-id/go-error-family in their module graph — disqualifying for private repos with hermetic nix vendorHash builds). Parser fixes ship ONCE, in `sseparse`; ssetest consumers get them through the re-exports. Release order matters: tag `sseparse/vX` first, then ssetest (which must carry a real `require sseparse vX` when its local replace is dropped at tag time). The zero-dependency property is guarded by `sseparse/go_mod_boundary_test.go`; the root must never require either nested module (`module_boundary_test.go`).
- **`datastartest` (go-datastar) delegates its SSE parsing to ssetest** — `datastartest/go.mod` requires `go-sse/ssetest`; its `reader.go` calls `ssetest.ReadEvents`/`ReadNEvents` and converts `[]ssetest.Event`, and re-wraps scan errors with its OWN error code (`datastartest.sse_scan_failed`), so the 2026-09-29 removal of `ssetest.CodeSSEScanFailed`/errorfamily from the parse path does not affect it until its pin is bumped — which the weekly `datastar-compat.yml` workflow exercises. It is NOT a single-dependency module (ssetest + go-sse land in its graph) — a future migration path to sseparse exists if that ever matters.
- **The parser is pinned to the official browser suite — and the corpus is DATA now.** `sseparse/testdata/wpt_format_corpus.json` is the single source of truth (29 vectors: WPT `eventsource/format-*`, spec § 9.2.6 examples, Chromium `event_source_parser_test.cc` cases, with upstream citations and notes). `sseparse.Corpus()`/`MustCorpus` load it for our own tests (`wpt_format_corpus_test.go`, `chunk_boundary_test.go`) AND for third-party parsers asserting against the same vectors without sseparse parsing in the loop. To add/change a vector, edit the JSON (not Go literals); `TestCorpusIntegrity` guards the artifact (floor 25 vectors, unique names, all three provenance families). `ssetest/roundtrip_test.go` + `FuzzWriteReadRoundTrip` close the loop against `sse.WriteEvent`. Any change to the reader or writer must keep corpus + chunk + round-trip green — they are the conformance contract, not ordinary tests.
- **`sseparse` line splitting is a custom `splitSSELines` bufio.SplitFunc** (CR, LF, CRLF — § 9.2.5 `end-of-line = cr lf / cr / lf`), NOT `bufio.ScanLines` (which only splits LF). A trailing buffered CR is held back until one more byte arrives so CRLF is always one terminator. Reader fixtures must serve bytes across multiple `Read` calls — the BOM probe consumes the first three bytes through its own read, so a one-shot reader fixture starves the scanner (see `partialReader` in `sseparse/reader_test.go`).
- **`sseparse` id/retry are sticky connection state, not per-frame fields** (spec § 9.2.6; Chromium `LastEventIdShouldNotBeReset`, `RetryTakesEffectEvenWhenNotDispatching`). An event's `ID`/`Retry` report the value in effect at dispatch; `id:` with NUL is ignored; empty `id:` resets to `""`; invalid `retry:` never resets a prior value. Exactly one leading UTF-8 BOM is stripped (a mid-stream BOM poisons a field name). An incomplete final frame at EOF is discarded.
- **The reader's line cap is `DefaultMaxLineBytes` (1 MiB), configurable per call** via `WithMaxLineBytes` (non-positive keeps the default; accepted by `ReadEvents`, `ReadNEvents`, `NewStreamReader`; the initial buffer never exceeds the cap so bufio enforces exactly the limit). Boundary semantics: the terminator must fit inside the cap too, so the largest parseable line is one byte UNDER the cap; at/over the cap the scan fails wrapping `bufio.ErrTooLong` (errors.Is-matchable). Pinned by `TestReadEvents_LineCapDefault` / `TestReadEvents_WithMaxLineBytes` in `sseparse/reader_test.go`.
- **A trailing LF in `sse.Event.Data` is structural, not payload**: `splitLines("x\n")` yields `["x"]` (no final empty line) and the spec strips one trailing LF from the data buffer at dispatch, so `"x\n"` and `"x"` are the same wire frame. Round-trip tests pin this.
- **ssetest `Collect*` closes bodies via `closeBody(tb, resp.Body)`** (2026-08-29): the bodyclose linter cannot see through the helper, so `.golangci.yml` excludes bodyclose for `ssetest/collect.go` (with the reason inline). Don't "fix" this by re-inlining `resp.Body.Close()` — that re-hides the Close-error branch that `collect_internal_test.go` covers via the erroring `io.ReadCloser` fake.
- **`nix flake check --all-systems` is green by declaration, not by luck**: the flake pins `systems` to `x86_64-linux`, `aarch64-linux`, `aarch64-darwin`. Nixpkgs 26.11 dropped x86_64-darwin — its derivations no longer evaluate. Re-add only after nixpkgs support returns.
- **`coverage-gate` flake app is hermetic**: declared runtimeInputs (go, bc, gnugrep) and its own `GOWORK=off`/`GOEXPERIMENT=jsonv2` exports. writeShellApplication PREPENDS runtimeInputs to the caller's PATH (verified via `bash -x`) — tools like `tail` resolve from the ambient environment, so apps work in a dev shell but can fail in minimal PATHs; declare any tool the script needs, and don't rely on ambient lookups.
- **gopls noise in this repo (documented, do not chase):** (1) the `stdversion` warnings on `encoding/json/v2` DISAPPEARED with the Go 1.27 directive bump (2026-09-19, normalized to `go 1.27` on 2026-09-29); if they return, the toolchain regressed below the directive. (2) residual "unnecessary type argument" hints — all inferable sites were cleaned 2026-08-29; the rest are required (`NewBroadcaster[int]()` has no args; `WithBufferSize[T]`'s T lives only in the return type). `gopls check` CLI reports 0. Note: a gopls/golangci-lint LSP running on the AMBIENT 1.26.7 toolchain reports a spurious `go.mod requires go >= 1.27` failure — run tools inside `nix develop`, whose `go_1_27` matches the directive; trust the CLI run over the stale editor diagnostic.
- **`testing/synctest` for the `CollectWithTimeout` tests: deliberately declined (2026-08-29).** synctest's own guidelines prohibit network I/O in a bubble; the `Collect*` helpers own real-socket httptest servers. A fake-net rewrite would test a different transport than production. Don't reopen without a fake-network design.
- **CI now covers what it used to miss (updated 2026-09-19):** examples build + `templ generate -check` (via `go run github.com/a-h/templ/cmd/templ@v0.3.1020` — `@version` form, because module-mode `go run` fails on templ's cmd deps missing from go.sum), `nix flake check` job (install-nix-action pinned by SHA; this is also the FORMAT gate — `checks.format` runs treefmt inside it), 6 fuzz targets, `govulncheck@v1.7.0` pinned (was `@latest`), and `scripts/smoke-examples.sh` (boots each example on PORT 18080/18765/18766 and requires real SSE bytes). The `.github/workflows/ci.yml` matrix runs 9 jobs — Test, Lint, Vet, Coverage, Examples, Example smoke, Nix flake check, Vulncheck, Fuzz — for both modules, under a cancel-in-progress concurrency group (plus `actionlint`, the scheduled `Flake update`, and the weekly `go-datastar compat` in their own workflows).
- **In-repo workflow permissions are not the whole story — repo-level GitHub settings can silently defeat automation.** The 2026-08-31 flake-update cron run pushed its branch, then `gh pr create` failed with "GitHub Actions is not permitted to create or approve pull requests" (repo Settings → Actions → Workflow permissions); the workflow's `|| echo "PR creation failed (likely already open)"` fallback swallowed it, so the step went green with no PR and an orphan branch. Fixed 2026-09-03: the repo setting `can_approve_pull_request_reviews` is now true, and the workflow checks for an existing PR explicitly so any real creation failure reddens the step. Lesson: never wrap `gh pr create` in an error-swallowing fallback, and remember repo settings live outside this repository — a fresh fork needs them re-enabled.
- **`vendorHash`/`vendorHashSsetest` must be recomputed whenever any `go.mod`/`go.sum` input or the replaced parent tree changes.** Flow: set the hash to `lib.fakeHash`, run `nix build`, copy the reported hash. Both hashes are independent (root vs ssetest module graphs); the `nix flake check` mismatch error reports the correct value directly. `sseparse` has NO vendorHash (`vendorHash = null`) — zero deps means no vendoring; a require sneaking in fails its hermetic check and `TestModuleStaysZeroDependency`.
- **`nolint` directives must LEAD the comment, and the annotated line must survive golines.** `// reason //nolint:dupword` (directive mid-comment) does not suppress; and if golines splits a single-line literal, a trailing `//nolint` gets detached from the flagged line — reviving the finding plus `nolintlint` violations. Keep annotated lines short enough that the formatter cannot split them.
- **Auto-git commit messages can be plausible-but-false — review them after the fact.** The background commit daemon writes messages from diffs it does not understand; commit `38e79aa` says "expand test coverage" for a deletion that _reduced_ coverage. After a session that deleted code or tests, run `git log --stat -5` and sanity-check the daemon's messages; published history stays unrewritten, but contradictions get recorded in the next status report so `git log` readers are not misled forever.
- **CI golangci-lint is pinned to the flake's version (v2.13.1) — keep them aligned.** A 2026-08-29 skew (CI v2.12.2 vs flake v2.13) let a `goconst` counting difference (2.12 counts test-file literals toward the threshold, 2.13 does not) pass locally and fail CI on every push. When bumping either side, bump both.
- **Status reports follow the conventions in [`docs/status/AGENTS.md`](docs/status/AGENTS.md)** — including the mandatory coverage-delta line (`- cover: library X% (+/-Δ), ssetest Y% (+/-Δ)`, measured this session, never quoted from memory). Read that file before writing any `docs/status/<date>_*.md` report; start from the copy-pasteable skeleton in [`docs/status/_template.md`](docs/status/_template.md).
- **Consumer-facing guides live in [`docs/guides/`](docs/guides/)** (EventStore patterns, filters and fan-out, reconnection and retry, DataStar SDK migration) and are linked from README's Guides section — when adding a guide, link it in both places.
- **`nix run .#coverage-gate` and `scripts/verify.sh` both survive a broken caller `GOCACHE`** (coverage-gate since 2026-09-19, verify.sh since 2026-09-29): the same mkdir -p probe falls back to a temp dir with a loud message, and coverage-gate no longer swallows `go test` stderr — a failing test run prints its log instead of dying as a silent exit 1 (the old behavior when a shell leaked `GOCACHE=/mnt/...`).
- **`scripts/verify.sh` runs a tidy-clean + directive-alignment gate (2026-09-29):** every module's `go.mod`/`go.sum` must be a `go mod tidy` no-op (snapshotted, restored on failure — the gate is read-only), and all three go directives must be identical. This is the gate form of the rule "`go mod tidy` is part of any go.mod edit, not a later chore". It also runs a bench smoke (`-bench=. -benchtime=1x`) so moved or broken benchmarks fail the gate instead of rotting silently.
- **Plain `go mod tidy` does not revert a hand-raised go directive** (tested 2026-09-29, correcting the earlier claim in the GOEXPERIMENT paragraph): the historical 1.27.1→1.27 reverts came from a normalize step. What catches a raised directive today is verify.sh's directive-equality check — three diverged directives redden the gate with a bump-them-together message.
- **The golangci-lint pin check lives in `scripts/golangci-pin.sh`, consumed twice:** verify.sh compares it against the LOCAL binary; the flake-update workflow compares it against the nixpkgs binary in the updated flake (`nix develop .#ci -c golangci-lint version`), so a nixpkgs golangci bump surfaces in the same weekly PR instead of redening the next human push (the 2.13.2→2.14.0 skew shipped silently through that workflow before this check existed).
- **`datastar-compat.yml` copes with the sseparse-carrying ssetest — verified 2026-09-29:** `go get ssetest@v0.4.0` (proxy-only, exact workflow commands on a go-datastar copy) auto-added `require .../sseparse v0.1.0 // indirect` to datastartest's go.mod and both go-datastar modules tested green. No pin-bump-set extension was needed; the module graph, not the workflow, enumerates dependencies.
- **`FuzzReadEvents` fuzzes the line cap too (2026-09-29):** its second int argument drives `WithMaxLineBytes`; the cap boundary (error wraps `bufio.ErrTooLong`) is pinned chunk-invariant. The committed seeds in `sseparse/testdata/fuzz/FuzzReadEvents/` carry the int dimension — new seeds need `int(<cap>)` lines, and hand-written seeds must satisfy `err == nil || errors.Is(err, bufio.ErrTooLong)`.
- **The corpus JSON is byte-canonical-gated:** `TestCorpusJSONIsCanonical` pins `testdata/wpt_format_corpus.json` to exactly `json.MarshalIndent(vectors, "", "  ")` plus a trailing newline, and `TestCorpusIntegrity` requires every vector to carry a `url` citation. Regenerate the file (decode via `Corpus()`, re-marshal, re-run conformance) instead of hand-formatting it.
- **`FuzzSplitSSELines` exercises the full production pipeline** (strip-one-BOM → splitter via `newSSEScanner`), not a bare `bufio.Scanner`: its reference model strips the leading BOM from the raw wire before comparing. Don't "simplify" `scanAllLines` back to a hand-rolled scanner — the consolidation exists so buffer-setup changes to `newSSEScanner` are fuzzed automatically.
- **`Corpus()` decode-failure paths are covered by mutating the package var** (`corpus_internal_test.go`): `corruptCorpusForTest` swaps `corpusJSON` and restores via `t.Cleanup`. The tests are deliberately NOT parallel — sequential tests complete before parked parallel tests resume, so the mutation cannot race the conformance readers. Keep that property if you touch them.

## Conventions

- **Allocation-free hot path:** `WriteEvent` uses direct byte appends (`buf = append(buf, 'e','v','e','n','t',':',' ')`) rather than `fmt.Fprintf`. `WriteRetry` uses `fmt.Fprintf` only because it's not hot. Preserve the append style when extending `WriteEvent`.
- **`//nolint:wrapcheck`** marks functions where raw errors are intentionally returned unwrapped because the underlying write error is already actionable (`WriteHeartbeat`, `WriteRetry`). `WriteEvent` _does_ wrap via `errorfamily.Wrapf` with the `Transient` severity.
- **Errors use `go-error-family`** with stable codes (`"sse.write_failed"`, `"sse.write_short"`, `"sse.send_failed"`, `"sse.event_id_invalid"`, `"sse.replay_failed"`, `"sse.replay_store_failed"`, `"sse.json_marshal_failed"`, `"sse.shutdown_failed"`, `"sse.shutdown_drain_deadline_exceeded"`) and severity categories (`Transient`, `Rejection`). Match this pattern for new errors.
- **Go 1.26 idioms in use:** integer range loops (`for i := range len(s)`, `for range 65`).
- **Tests:** external package `sse_test`, every test starts with `t.Parallel()`, `httptest` for stream tests, `bytes.Buffer` for serialization, `errorWriter`/`errorResponseWriter` fakes for failure paths. Race tests exist (`TestBroadcaster_BroadcastUnsubscribeRace`).
- **`sseparse` and `ssetest` public helpers take `tb testing.TB`** (never `*testing.T`) so they work with `*testing.B` and Ginkgo's `GinkgoT()`; the `thelper` linter enforces the param name `tb`. The root `sse` package must never require `ssetest` or `sseparse` (module boundary, guarded by `module_boundary_test.go`); `ssetest` reaches both via tagged module requires.
- **`splitLines`** (event.go) handles the SSE multi-line `data:` requirement and CRLF stripping; its zero-alloc core `forEachLine` walks lines via a callback (the hot path `WriteEvent` uses — 1 alloc/op for a simple event, verified 2026-09-19), while `splitLines` collects into a slice for callers that need one. The old "fast path allocates nothing" claim was false — `[]string{s}` always allocated one backing array.
- **`KeyedLines`** (event.go) prefixes each line of a multi-line value with `key`, producing the newline-joined string for `Event.Data`. This is the building block for keyed-data-line protocols like DataStar (`data: elements <div>` / `data: elements </div>`). It delegates to `splitLines` for line splitting. Returns `""` for empty input (no data line emitted).
- **`Stream.SendLines`** (stream.go) is a convenience wrapper around `Send` that joins variadic string arguments with `\n` into `Event.Data`. Composes with `KeyedLines` for the DataStar pattern: each `KeyedLines` result (which may contain embedded `\n`) is one argument, and `splitLines` in `WriteEvent` handles the final split.
- **`WriteKeyedLines`** (event.go) is the wire-only single-key convenience: `WriteEvent(w, Event{Event: eventType, Data: KeyedLines(key, value)})`. For consumers that use `WriteEvent` directly without a `Stream`.
- **`Stream.SendKeyed`** (stream.go) is the stream-level single-key convenience: `Send(Event{Event: eventName, Data: KeyedLines(key, value)})`. For the common single-key DataStar pattern (e.g., `patch-signals`).

## Examples (`example/`)

Three runnable examples, each an independent `package main`:

- `example/server.go` — raw wire-format demo (curl-driven broadcast/fan-out), port `:8080`.
- `example/datastar/` — DataStar activity feed (fan-out, filtering, replay, heartbeat), port `:8765`.
- `example/htmx/` — HTMX HTML-fragment-swap UI, port `:8766`.

**`example/README.md` compares the DataStar vs HTMX approaches in detail** (mechanism, payload size, granularity, bundle size, trade-offs). The two browser examples have different scopes:

- **DataStar** (`example/datastar/`) is a **live activity feed** that demonstrates every go-sse feature: `Broadcaster` fan-out (open multiple tabs), `SubscriberCount` (live counter via `OnSubscribe`/`OnUnsubscribe`), `SubscribeFilter` (`?filter=alerts`), `EventStore` + `Replay` (close/reopen tab), and `Heartbeat` (proxy keep-alive). A background producer goroutine emits events every 2s; each handler subscribes, replays missed events on reconnect, and forwards live events to the `Stream`. Uses `KeyedLines`/`SendKeyed`/`SendLines` for DataStar's keyed-data-line wire format (`datastar-patch-signals` for reactive state, `datastar-patch-elements` for DOM patches).
- **HTMX** (`example/htmx/`) is a simpler progress-bar demo that uses plain `stream.Send(Event{...})` to stream HTML fragments; the HTMX SSE extension (`sse-swap="progress"` + `hx-swap="innerHTML"`) swaps them into the DOM. No HTMX-specific helpers are needed — HTMX speaks vanilla SSE.

**Shared structure (both browser examples):**

- `index.templ` — type-safe HTML template (templ). Run `templ generate` after editing.
- `index_templ.go` — generated code (checked into git, excluded from treefmt and golangci-lint via `*_templ.go` path patterns).
- `static/styles.css` — CSS with dark/light theme support via `prefers-color-scheme` and manual `data-theme` toggle.
- `static/*.js` — client JS bundles, embedded via `go:embed` (no CDN).

**DataStar example file split** (`example/datastar/`):

The DataStar example is split across four Go files for readability:

- `main.go` — constants, `go:embed`, and `main()` (server setup, signal handling, graceful shutdown).
- `store.go` — `memStore`: in-memory ring-buffer `EventStore` for reconnection replay.
- `producer.go` — `activityItem` struct, message templates, event builders (`feedItemEvent`, `countEvent`, `replayEvent`, `totalEventSignal`), `startProducer` goroutine.
- `handlers.go` — `activityServer` struct, `newActivityServer` (broadcaster + store wiring with OnSubscribe/OnUnsubscribe), `indexHandler`, `eventsHandler` (CORS, replay, SubscribeFilter, heartbeat, event loop).
- `static/app.js` — theme toggle (localStorage persistence), keyboard shortcuts (`a`/`e`), smart scroll-to-top on new feed items.
- `main_test.go` — unit tests (memStore replay/ring-buffer), integration tests (fan-out, subscriber count, graceful shutdown, filter predicate, CORS header, `$replayed` reset).
- `VERIFY.md` — 2-minute browser verification checklist.

**HTMX** (`example/htmx/`) uses a single `main.go`.

**Running:** `go run ./example/datastar/` or `go run ./example/htmx/` (run the package, not a single file, because the page render lives in the generated `index_templ.go`).

**Vendored client bundles:**

- DataStar: `static/datastar.js` (DataStar v1.0.2).
- HTMX: `static/htmx.min.js` (htmx 2.0.4) + `static/sse.min.js` (htmx-ext-sse 2.2.4). HTMX needs the separate SSE extension for `sse-connect`/`sse-swap`.

**Editing the template:** After changing `index.templ`, run `templ generate` to regenerate `index_templ.go`. The `templ` CLI is in the devShell.

**HTMX restart mechanism:** HTMX SSE connections are declared in markup (`sse-connect="/events"`). The Restart button fetches a fresh `#sse-container` fragment (`GET /sse-container`) and swaps it in (`outerHTML`), which tears down the old `EventSource` and opens a new one — no JavaScript. More ceremony than DataStar's one-attribute `@get('/events')`, but idiomatic HTMX.

## What This Library Is NOT

No CQRS, no dashboard/routes/templates, no WebSockets, no event bus, no payload-format opinion. Adding any of these breaks the library's scope. Consumers build domain layers on top.

**No retry/backoff loop** (evaluated 2026-08-07, declined):

- go-sse has no retryable operation: a failed `Stream.Send` means a dead or partially-written connection — retrying re-emits bytes already on the wire and corrupts SSE frames. `EventStore` retry policy belongs to the consumer's store; the `Shutdown` drain is a condition-wait; `Broadcast`'s drop-on-full is invariant #2.
- **Trap:** connection-death errors (`sse.send_failed`, `sse.write_failed`, `sse.replay_failed`) are `Transient`, and `errorfamily.IsRetryable` retries exactly `Transient` — a default-configured retry wrapper would retry a broken pipe. Here `Transient` means "drop the connection and let the browser reconnect", not "try the same socket again".
- `Event.Retry`/`WriteRetry` already own the word "retry" here with an unrelated meaning (a browser reconnect hint in ms).

Full analysis and re-open triggers: [docs/brainstorming/2026-08-07_go-retry-adoption-evaluation.md](docs/brainstorming/2026-08-07_go-retry-adoption-evaluation.md).

## Companion Library: go-datastar

[`go-datastar`](https://github.com/LarsArtmann/go-datastar) is a standalone DataStar protocol library built on go-sse. It provides `Patch` interface (`Event() sse.Event`) with `ElementsPatch`, `SignalsPatch`, `ScriptPatch`, `RedirectPatch`, etc. as first-class values — storable, filterable, replayable, broadcastable via `sse.Broadcaster[sse.Event]`.

go-sse deliberately has no DataStar-specific types or event-name constants — it remains a transport library. The `JoinLines`, `KeyedLines`, `SendLines`, `SendKeyed`, and `WriteKeyedLines` helpers are general SSE primitives that DataStar happens to consume. Absorbing DataStar patch types would turn a clean transport library into a protocol-coupled one.
