# AGENTS.md — go-sse

Server-Sent Events transport library for Go. Single package (`sse`), flat layout, plus two nested modules: `sseparse/` (zero-dependency parser + assertions + WPT corpus-as-data) and `ssetest/` (HTTP collection helpers layered on sseparse via re-exports). Wire format + connection lifecycle + fan-out + reconnection replay. Two small root dependencies (`go-branded-id`, `go-error-family`); no domain opinions.

## Commands

`flake.nix` provides hermetic build/test/lint. The devShell sets `GOWORK=off` automatically:

```bash
nix run .#test-race          # tests with race detector (root + ssetest + sseparse)
nix run .#vet                # go vet (root + ssetest + sseparse)
nix run .#lint               # golangci-lint (root + ssetest + sseparse)
nix run .#coverage           # test + coverage report (root measured as package `.` — never ./...,
                            #   the 0%-covered example packages dilute the number to ~59%)
nix run .#coverage-gate      # fail under ratchet thresholds (library 100.0%, ssetest 100.0%, sseparse 100.0%)
scripts/verify.sh            # one-command pre-push gate (fmt+vet+lint+test+flake check; --fast skips flake)
                            #   --fast is NOT sufficient for go.mod/go.sum/flake.nix changes: the skipped
                            #   flake check is the only gate that catches vendorHash drift — pair it.
nix flake check              # full hermetic check (compile + test, root + ssetest + sseparse)
nix develop                  # enter dev shell (Go 1.27, golangci-lint, gopls, govulncheck, templ, lychee, vulnix)
```

Formatting via treefmt (`nix fmt`): gofumpt, goimports, golines, nixfmt, shfmt. Generated `*_templ.go` files are excluded from treefmt and golangci-lint. Only `nix fmt` output is committable — CI's `checks.format` rebuilds the same check, and any hand-rolled or daemon-produced "reformat" that differs from it reddens master.

Raw `go` tooling (for environments without Nix):

```bash
GOWORK=off go test ./... -race -count=1   # tests
GOWORK=off go vet ./...                   # vet
GOWORK=off golangci-lint run ./...        # lint
(cd ssetest && GOWORK=off go test ./... -race -count=1)  # ssetest module
(cd sseparse && GOWORK=off go test ./... -race -count=1) # sseparse module
templ generate                            # regenerate *_templ.go from .templ files
```

`ssetest/` and `sseparse/` are their own Go modules (`github.com/larsartmann/go-sse/ssetest` / `.../sseparse`), with no `replace` directives: ssetest consumes `go-sse` v0.6.2 and `sseparse` v0.2.1 as tagged modules, so a local, untagged sseparse change is only visible to ssetest tests via a temporary local `replace => ../sseparse` (revert before committing) or a fresh sseparse tag. sseparse has ZERO third-party requires (test-guarded by `sseparse/go_mod_boundary_test.go` — the entire reason it exists). Always `cd` into each module to build/test/lint it; `go test ./ssetest/...` from the root fails (module boundary). golangci-lint discovers the root `.golangci.yml` from inside both nested modules automatically.

**Go directive policy: minor form (`go 1.27`), all three modules identical.** Bump all three `go.mod` files TOGETHER — `scripts/verify.sh` fails the gate when they diverge, and plain `go mod tidy` does not align a lone drift. Deliberately minor form, NOT patch-pinned (fleet de-poisoning, 2026-10-03): a patch-pinned floor re-poisons every consumer's own directive through MVS selection — libraries declare minor form. Proxy-cached tags keep their committed go.mod, so re-tagging is the only way to change a released directive. `GOEXPERIMENT=jsonv2` was removed everywhere (2026-09-29; unnecessary on Go 1.27): do not re-add it, `scripts/verify.sh` grep-guards it.

**`GOWORK=off` is required in this environment** (a parent `go.work` includes a sibling with a stale checksum → `SECURITY ERROR` under workspace mode). It is gitignored and unneeded in a fresh clone; the devShell and `.envrc` set it automatically.

**`buildflow` needs direnv (`.envrc`), not just the devShell** — it is a system binary that inherits the ambient shell and ignores both the flake devShell and `.buildflow.yml` env config. The gitignored `.envrc` (`use flake` + `export GOWORK=off`) is what propagates the toolchain and `GOWORK=off`. Symptom of a missing/un-allowed `.envrc`: an old ambient toolchain failing with `go.mod requires go >= X`, or a workspace-mode `SECURITY ERROR`.

## Dependencies

Two library dependencies, both `github.com/larsartmann/*`: `go-branded-id` (phantom-type branded IDs, `EventID = brandid.ID[eventBrand, string]`) and `go-error-family` (structured error wrapping with severity categories). One example-only dependency: `a-h/templ` for the browser examples — the library never imports it; the generated `*_templ.go` files are checked in, so consumers never need the templ CLI.

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

`Broadcaster[T]` is a thin public wrapper (`struct{ *fanOut[T] }`); all real logic lives in the unexported `fanOut[T]` in `fanout.go` (transport-agnostic, reusable). When adding broadcaster behavior, edit `fanout.go`, not `broadcaster.go`.

## Concurrency Invariants (critical)

1. **`Stream.mu` serializes ALL writes** to the underlying `ResponseWriter` (`Send`, `Heartbeat`, `Close` all acquire it). Any new write path must hold this mutex. `Close` sets `closed` under the mutex and `Heartbeat` checks it before writing — a tick racing handler teardown must return, not flush a finished `ResponseWriter` (that race panicked inside `net/http`; pinned by `TestStream_HeartbeatStopsAfterClose`).
2. **`fanOut` uses RWMutex with non-blocking sends.** `Broadcast` holds `RLock` during iteration and drops on full buffers via `select`/`default`. The read lock guarantees `Unsubscribe` cannot close a channel mid-send — never make Broadcast blocking under it. **SubscribeFilter predicates run under this same read lock**: they must be pure, fast, non-blocking.
3. **`fanOut.Close()` sets `subscribers = nil` as the closed sentinel.** `Subscribe` checks for nil and returns an already-closed channel. Don't repurpose nil to mean "uninitialized."
4. **`fanOut.draining` is the "shutdown in progress" sentinel** (distinct from closed) so `Subscribe` keeps returning closed channels during a drain. Set under the write lock, check under either.
5. **`fanOut.onDrop` runs inside the read lock in `sendAllLocked`**, on the broadcasting goroutine, per subscriber with a full buffer (nil-guarded). It must be fast and non-blocking; `safeDropCall` recovers panics and continues. Re-entrant broadcaster calls from the callback still deadlock (read vs write lock).

## Lifecycle API

- `Close()` — instant shutdown; closes all subscriber channels immediately.
- `Shutdown(ctx)` — graceful drain; rejects new `Subscribe` calls, waits for every subscriber's buffer to empty, then closes channels. Deadline fires → the context's error wrapped via `errorfamily` (`sse.shutdown_drain_deadline_exceeded`), with `errors.Is(err, context.DeadlineExceeded/Canceled)` preserved. Safe from multiple goroutines; only the first drain takes effect. Polls at 1ms (`waitForDrain`); a genuinely slow subscriber keeps it waiting until the context fires.
- `Health() BroadcasterHealth` — cheap value snapshot (`Closed`, `Draining`, `SubscriberCount`, `BufferSize`); suitable for k8s probes.
- `NewBroadcaster[T](opts ...Option[T])` — `WithBufferSize[T]` (read once; non-positive silently keeps the `DefaultSubscriberBuffer` = 64 default) and `WithOnDrop[T]`; `OnDrop(fn)` is the runtime setter (nil clears).

## Gotchas

- **`.buildflow.yml` currently skips nothing** — the 2026-09-29 `go-structure-linter` skip was UNSKIPPED 2026-10-09 (linter v0.11.3 auto-defers root-package-files for flat modules). The file carries deliberate WONT-adopt policy notes (sseparse stays on encoding/json v1; `lo.Filter` declined — samber/lo is not a dep). If go-structure-linter findings reappear, they belong in this repo's own `.go-structure-linter.yaml`, not a skip; don't "fix" flat-layout findings by moving files.
- **golangci-lint is deliberately NOT pinned in `flake.nix`** (nixpkgs ships one floating package). The effective version is pinned by `flake.lock`; skew is surfaced twice — `scripts/golangci-pin.sh` is consumed by verify.sh's pin-check (local binary vs the ci.yml pin) and by the flake-update workflow's cross-check (nixpkgs binary vs the pin, in the same weekly PR). When a flake update moves golangci-lint, bump ci.yml's `version:` pin in that PR.
- **`NewStream` writes `200 OK` immediately** and sets headers — it is not lazy. Do not write to `w` before `NewStream`.
- **`flusher` is an unexported interface** (`interface{ Flush() }`). `NewStream`'s `w.(flusher)` silently yields `nil` for non-flushing writers; `Send`/`Heartbeat` nil-check before flushing.
- **Broadcast is intentionally lossy** (`DefaultSubscriberBuffer` = 64/subscriber; overflow drops). Guaranteed delivery is the consumer's job (ack/replay). Do not "fix" into a blocking send — head-of-line blocking.
- **`onDrop` fires per-subscriber, not per-message**: one `Broadcast` to N full buffers invokes it N times with the same message (also per message in `BroadcastMany`).
- **`MustParseEventID` panics** — tests and constants only. Use `ParseEventID` (rejects NUL, `\n`, `\r` — the spec § 9.2.4 Last-Event-ID value space) for request-header values. Empty `EventID` is the zero/initial-connection value, not an error; `Replay` with it replays everything.
- **Strong-ID policy (2026-10-11): IDs are `sse.EventID` at every Go boundary** — `ssetest.WithLastEventID` (zero value sends no header, browser initial-connection behavior, pinned by `TestCollect_WithLastEventID_ZeroIDSendsNoHeader`) and the DataStar example's `feedItemEvent`. Deliberately raw: `sseparse`'s `Event.ID`, `ExpectedEvent.ID`, `lastID` — branding there would add sseparse's first third-party require and break the ssetest re-exports/datastartest; validation belongs to the transport boundary (`sse.ParseEventID`). Suppressed in-source with the native `//branching-flow:ignore strong-id` directive plus a reason line above it — NOT `//nolint:branching-flow`, which makes golangci warn about unknown linters on every run. Default tool output: 0 rows; `--include-suppressed` lists the deliberate skips.
- **`OnDisconnect` callbacks fire inside `Close()`**, after the mutex is released, in registration order.
- **Predicate panics are recovered and treated as non-matches** (`safePredCall`): one broken predicate skips the event for that subscriber only. Same contract for `onDrop` (`safeDropCall`). `ReplayFiltered` type-asserts to `FilteredEventStore` (predicate pushed into the store query) and otherwise falls back to `EventsAfter` + post-filter; nil predicate delegates to `Replay`. Internal `subscriber[T]` wraps `chan T` + optional predicate; the map is keyed by channel pointer identity.
- **`sseparse` is the parser; `ssetest` re-exports it** (parser consumers must not pay go-sse's module graph — disqualifying for private repos with hermetic nix vendorHash builds). Parser fixes ship ONCE in `sseparse`. Release order: tag `sseparse/vX` first, then ssetest. The root must never require either nested module (`module_boundary_test.go`).
- **`datastartest` (go-datastar) delegates SSE parsing to ssetest** (`ReadEvents`/`ReadNEvents`), re-wrapping scan errors with its own code; it pins ssetest v0.4.0 and does NOT use ssetest's `WithLastEventID` (it has its own string-typed option), so ssetest breaking changes don't ripple into it unless they touch the parse path.
- **The parser is pinned to the official browser suite, and the corpus is DATA.** `sseparse/testdata/wpt_format_corpus.json` (29 vectors with upstream citations) is the single source of truth, loaded by `Corpus()`/`MustCorpus` for our tests AND third-party parsers. Add vectors only via the ingestion generator (`go generate` in sseparse; procedure: `sseparse/README.md` §"Ingesting new WPT vectors") — hand-editing fights the byte-canonical gate (`TestCorpusJSONIsCanonical` pins exact `MarshalIndent` + trailing newline; its failure message points at the procedure). `FuzzReadEvents` fuzzes the line cap (seeds carry the int dimension; seeds must satisfy `err == nil || errors.Is(err, bufio.ErrTooLong)`); `FuzzSplitSSELines` exercises the full production pipeline. Corpus/chunk/round-trip tests are the conformance contract, not ordinary tests — keep them green. Living account: [`docs/conformance.md`](docs/conformance.md).
- **`sseparse` line splitting is a custom `splitSSELines` SplitFunc** (CR, LF, CRLF), not `bufio.ScanLines` (LF only); a trailing buffered CR is held back one byte so CRLF stays one terminator. Reader fixtures must serve bytes across multiple `Read` calls (the BOM probe starves one-shot readers — see `partialReader`).
- **`sseparse` id/retry are sticky connection state** (spec § 9.2.6): events report the value in effect at dispatch; NUL `id:` ignored; empty `id:` resets; invalid `retry:` never resets. One leading BOM stripped (mid-stream BOM poisons a field name); incomplete final frame at EOF discarded. The line cap is `DefaultMaxLineBytes` (1 MiB), configurable via `WithMaxLineBytes`; the terminator must fit too, so the largest parseable line is one byte under the cap — at/over fails wrapping `bufio.ErrTooLong`.
- **A trailing LF in `Event.Data` is structural, not payload** — `"x\n"` and `"x"` are the same wire frame (round-trip tests pin this).
- **ssetest `Collect*` closes bodies via `closeBody(tb, resp.Body)`** — the bodyclose linter cannot see through the helper, so `.golangci.yml` excludes bodyclose for `ssetest/collect.go` (reason inline). Don't re-inline `resp.Body.Close()`.
- **`nix flake check --all-systems` is green by declaration**: `systems` is pinned to x86_64-linux, aarch64-linux, aarch64-darwin (nixpkgs 26.11 dropped x86_64-darwin evaluation; re-add only when support returns).
- **Flake apps are hermetic**: declare every tool in `runtimeInputs` (writeShellApplication prepends them to PATH; ambient lookups fail in minimal environments). Both `coverage-gate` and `verify.sh` survive a broken caller `GOCACHE` (mkdir probe → loud temp-dir fallback); coverage-gate never swallows `go test` stderr.
- **gopls noise (do not chase):** residual "unnecessary type argument" hints are required explicitness (`NewBroadcaster[int]()`, `WithBufferSize[T]`'s T in return position only). An LSP on an ambient toolchain older than the directives reports a spurious `go.mod requires go >= X` — run tools inside `nix develop` and trust the CLI over the editor.
- **CI covers the gates that used to be local-only** (11 jobs: Test, Lint, Vet, Coverage, Coverage gate, Examples, Smoke, Shellcheck, Nix flake check, Vulncheck, Fuzz; plus actionlint, scheduled Flake update, weekly go-datastar compat): examples build + `templ generate -check` (`go run ...@v0.3.1020` — `@version` form; module-mode fails), the Coverage gate at the flake app's 100.0/100.0/100.0 thresholds (the checked-in up-only ratchet baseline, `docs/benchmarks/coverage-baseline.txt`), Shellcheck, the Nix job (also the FORMAT gate via `checks.format`), a smoke job requiring real SSE bytes, and govulncheck. Runners are pinned to `ubuntu-24.04` (GitHub migrates `ubuntu-latest` to Ubuntu 26 from 2026-10-19; flip back only after every job is verified green on the new image). `ci.yml` triggers on master pushes and PRs ONLY — module tags never run it, so release verification is `scripts/release-verify.sh`'s job.
- **`vendorHash`/`vendorHashSsetest` must be recomputed whenever any `go.mod`/`go.sum` input or the replaced parent tree changes**: set `lib.fakeHash`, `nix build`, copy the reported hash. The two hashes are independent; `sseparse` has none (zero deps — a require sneaking in fails its hermetic check).
- **`nolint` directives must LEAD the comment**, and the annotated line must survive golines (a split literal detaches a trailing `//nolint`, reviving the finding plus nolintlint violations). Keep annotated lines short.
- **Auto-git daemon commits are plausible-but-false and can be non-compiling — review, don't trust.** Commit surgically when the message matters; after any daemon sweep touching `.go` files run `go vet ./...` before trusting the tree; sanity-check `git log --stat -5` after deletion-heavy sessions. Known root-caused trap kept closed: BuildFlow's brandid-lint repair stub only accepts a literal-returning `Name()` — `eventBrand.Name()` returns the literal `"SSEEvent"` and must stay a literal.
- **Actions-created PRs run CI only after human approval** (`action_required` for GITHUB_TOKEN PRs). Never mistake a missing run for a green one, and never merge one while master is red (PR #7 merged into a red tree, 2026-10-10). Policy tracked in TODO_LIST Blocked.
- **Repo-level GitHub settings can silently defeat automation** — a workflow can be green while its `gh pr create` was denied (the 2026-08-31 orphan branch). Never wrap `gh pr create` in an error-swallowing fallback.
- **flake.lock rebase conflicts resolve to the strictly-newer union — never hand-merge JSON**: compare per-input `lastModified`, take the newer per axis, and prove tree identity before pushing (`git diff HEAD <old-tip>` empty).
- **`git town sync` needs a TTY**; replicate manually: `git -c rebase.updateRefs=false rebase origin/master`, resolve, `GIT_EDITOR=true git rebase --continue`, push.
- **Probe CI health at session start** (`gh run list --branch master --limit 5`) — master has repeatedly sat red while local trees looked clean.
- **Status reports follow [`docs/status/AGENTS.md`](docs/status/AGENTS.md)** (mandatory measured-this-session coverage-delta line; skeleton in `docs/status/_template.md`). Consumer-facing guides live in [`docs/guides/`](docs/guides/) and are linked from README — link new guides in both places.
- **Coverage-gate is toolchain-sensitive: measure with the current lock, cover branches, never pad thresholds.** A Go patch bump shifts statement attribution by tenths of a point; the up-only ratchet baseline (`docs/benchmarks/coverage-baseline.txt`) replaced zero-margin floats — regressions redden with a fix-or-lower-deliberately message, improvements get a ratchet-up hint.
- **`verify.sh` runs a tidy-clean + directive-alignment gate** (every module's `go.mod`/`go.sum` must be a `go mod tidy` no-op; all three directives identical) plus a bench smoke, an AGENTS-facts assertion (documented thresholds/directive policy vs the tree), and a bare-shfmt fingerprint guard over `scripts/*.sh` (the drift class that reddened master twice as a "formatting" daemon commit).
- **`datastar-compat.yml` copes with the sseparse-carrying ssetest**: `go get ssetest@vX` auto-adds the sseparse require; the module graph, not the workflow, enumerates dependencies (verified 2026-09-29).
- **`Corpus()` decode-failure tests mutate the package var** (`corruptCorpusForTest` + `t.Cleanup`) and are deliberately NOT parallel — sequential tests complete before parked parallel tests resume. Keep that property.

## Conventions

- **Allocation-free hot path:** `WriteEvent` uses direct byte appends, not `fmt.Fprintf` (`WriteRetry` may, it's not hot). Preserve the append style. `WriteEvent` wraps errors via `errorfamily.Wrapf` (`Transient`); `WriteHeartbeat`/`WriteRetry` intentionally return raw errors (`//nolint:wrapcheck`).
- **Errors use `go-error-family`** with stable codes (`sse.write_failed`, `sse.write_short`, `sse.send_failed`, `sse.event_id_invalid`, `sse.replay_failed`, `sse.replay_store_failed`, `sse.json_marshal_failed`, `sse.shutdown_failed`, `sse.shutdown_drain_deadline_exceeded`) and severities (`Transient`, `Rejection`). Note: `Transient` here means "drop the connection, let the browser reconnect" — a default retry wrapper would retry broken pipes (see "What This Library Is NOT").
- **Go 1.27 idioms:** integer range loops (`for i := range len(s)`).
- **Tests:** external package `sse_test`, `t.Parallel()` everywhere, `httptest` for streams, `bytes.Buffer` for serialization, `errorWriter` fakes for failure paths. sseparse/ssetest public helpers take `tb testing.TB` (thelper enforces the name).
- **`splitLines`/`forEachLine`** (event.go): the callback core powers the hot path (1 alloc/op); `splitLines` collects a slice for callers that need one. **`KeyedLines`** prefixes each line of a multi-line value with `key` (returns `""` for empty input); **`SendLines`** joins variadic args with `\n`; **`SendKeyed`**/`WriteKeyedLines` are the single-key conveniences (DataStar's keyed-data-line pattern).

## Examples (`example/`)

Three runnable `package main` demos: `example/server.go` (raw wire format, :8080), `example/datastar/` (live activity feed exercising every feature — fan-out, subscriber count, filtering, replay, heartbeat, keyed wire format; :8765), `example/htmx/` (progress-bar fragment swap; :8766). [`example/README.md`](example/README.md) compares the DataStar vs HTMX approaches in detail.

Structure: `index.templ` + generated `index_templ.go` (run `templ generate` after editing; CLI in the devShell), `static/` with hand-written CSS/JS and vendored client bundles (DataStar v1.0.2; htmx 2.0.4 + htmx-ext-sse 2.2.4) embedded via `go:embed`. DataStar splits across `main.go`/`store.go`/`producer.go`/`handlers.go` with `main_test.go` + a 2-minute `VERIFY.md` browser checklist. Run packages, not single files: `go run ./example/datastar/`. HTMX restarts via a fragment swap that tears down the old `EventSource` — no JS.

## What This Library Is NOT

No CQRS, no dashboard/routes/templates, no WebSockets, no event bus, no payload-format opinion. Adding any of these breaks the library's scope.

**No retry/backoff loop** (evaluated, declined): a failed `Stream.Send` means a dead or partially-written connection — retrying re-emits bytes and corrupts frames. Connection-death errors are `Transient`, and `errorfamily.IsRetryable` retries exactly `Transient` — a default retry wrapper would retry a broken pipe. `Event.Retry`/`WriteRetry` already own the word with an unrelated meaning (browser reconnect hint). Full analysis: [docs/brainstorming/2026-08-07_go-retry-adoption-evaluation.md](docs/brainstorming/2026-08-07_go-retry-adoption-evaluation.md).

## Companion Library: go-datastar

[`go-datastar`](https://github.com/LarsArtmann/go-datastar) is a standalone DataStar protocol library built on go-sse: `Patch` interface (`Event() sse.Event`) with `ElementsPatch`, `SignalsPatch`, etc. as storable, filterable, replayable, broadcastable values. go-sse deliberately has no DataStar-specific types — the keyed-line helpers are general SSE primitives DataStar happens to consume; absorbing patch types would make a clean transport library protocol-coupled.
