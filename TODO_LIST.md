# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, use ROADMAP.md.
> Items are ranked by impact. Status is verified, not assumed.

## Status legend

| Status           | Meaning                                                     |
| ---------------- | ----------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                                   |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                   |
| 🔵 `BLOCKED`     | Cannot proceed, external dependency or decision needed.     |
| 🟢 `DONE`        | Completed. Remove from this list and log in `CHANGELOG.md`. |

## High Impact

| Task                                                                       | Status    | Impact | Effort | Evidence                                                           |
| -------------------------------------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------------------ |
| Create `flake.nix` with devShell (test, lint, vet) per LarsArtmann convention | 🔴 `TODO` | High   | 1h     | No flake.nix exists; global AGENTS.md mandates it; `go-branded-id` has one |
| Create CI workflow (`.github/workflows/ci.yml`): test + lint + vet + race  | 🔴 `TODO` | High   | 1h     | `.golangci.yml` configured but nothing runs it automatically      |

## Medium Impact

| Task                                                                       | Status    | Impact | Effort | Evidence                                                           |
| -------------------------------------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------------------ |
| Add fuzz tests for `WriteEvent` (serializer) and `ParseEventID` (validator) | 🔴 `TODO` | Med    | 2h     | `event.go:96`, `event.go:44` — parser/serializer are prime fuzz targets |
| Add integration test with real `http.Server` (not just `httptest.NewRecorder`) | 🔴 `TODO` | Med    | 1h     | All stream tests use `httptest.NewRecorder`; no real server test   |
| Add test for `Broadcast` after `Close` on `Broadcaster`                    | 🔴 `TODO` | Med    | 30min  | `fanout.go:116` — close-then-broadcast path not explicitly tested |
| Add test for double-`Close()` safety on `Stream`                           | 🔴 `TODO` | Med    | 30min  | `stream.go` — no test guards against double-close                  |
| Add build-tags note to CONTRIBUTING.md lint instructions                   | 🔴 `TODO` | Med    | 10min  | `.golangci.yml` declares `goexperiment.jsonv2` tag; CONTRIBUTING.md does not mention it |

## Low Impact

| Task                                                                       | Status    | Impact | Effort | Evidence                                                           |
| -------------------------------------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------------------ |
| Replace recursive `contains()`/`startsWith()` with `strings.Contains`      | 🔴 `TODO` | Low    | 10min  | `stream_test.go:317-334` — hand-rolled O(n²) recursive helpers     |
| Replace custom `itoa()` with `strconv.Itoa`                                | 🔴 `TODO` | Low    | 5min   | `broadcaster_test.go:414` — reinvents stdlib                      |
| Verify `errorResponseWriter` nil-pointer path in `replay_test.go`          | 🔴 `TODO` | Low    | 15min  | `replay_test.go:152-161` — `writer` field nil-initialized           |
| Add test for unicode/special chars in `EventID`                            | 🔴 `TODO` | Low    | 15min  | `event.go:44` (`ParseEventID`) — no unicode edge-case test          |
| Add `pkg.go.dev` reference URL once published                              | 🔴 `TODO` | Low    | 5min   | README has no doc link; module not yet tagged for publication       |
| Add coverage reporting to CI                                               | 🔴 `TODO` | Low    | 20min  | No coverage step in any config                                     |
| Add `example/` directory with runnable examples                           | 🔴 `TODO` | Low    | 30min  | No examples; README snippets are inline only                       |
