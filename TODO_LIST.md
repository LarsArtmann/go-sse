# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, use ROADMAP.md.
> Items are ranked by Pareto impact. Status is verified, not assumed.

## Status legend

| Status           | Meaning                                                     |
| ---------------- | ----------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                                   |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                   |
| 🔵 `BLOCKED`     | Cannot proceed, external dependency or decision needed.     |
| 🟢 `DONE`        | Completed. Remove from this list and log in `CHANGELOG.md`. |

## High impact — reliability and correctness

| Task                                                                                               | Status    | Impact | Effort | Evidence                                                                         |
| -------------------------------------------------------------------------------------------------- | --------- | ------ | ------ | -------------------------------------------------------------------------------- |
| Replace `time.Sleep(200ms)` in `TestIntegration_BroadcasterFanOut` with deterministic channel sync | 🔴 `TODO` | High   | 30min  | `integration_test.go:111` — race condition masquerading as a test                |
| Replace `time.Sleep(50ms)` in `TestStream_Heartbeat` with deterministic sync                       | 🔴 `TODO` | High   | 20min  | `stream_test.go:152` — flaky under scheduler pressure                            |
| Add heartbeat delivery integration test (real HTTP round-trip)                                     | 🔴 `TODO` | High   | 45min  | No heartbeat test in `integration_test.go`; most critical proxy-survival feature |
| Add Last-Event-ID reconnection replay integration test                                             | 🔴 `TODO` | High   | 45min  | No replay test in `integration_test.go`; reconnection is the core SSE use case   |

## Medium impact — coverage gaps and CI hardening

| Task                                                                                                               | Status    | Impact | Effort | Evidence                                                                     |
| ------------------------------------------------------------------------------------------------------------------ | --------- | ------ | ------ | ---------------------------------------------------------------------------- |
| Close 4 coverage gaps: `Name` 0%, `MustParseEventID` success 75%, `splitLines` 95.5%, `Heartbeat` error path 91.7% | 🔴 `TODO` | Med    | 1h     | `go tool cover -func` shows these 4 below 100%; overall 97.9%                |
| Add `govulncheck` job to CI                                                                                        | 🔴 `TODO` | Med    | 20min  | devShell includes it; `.github/workflows/ci.yml` does not run it             |
| Add fuzz job to CI (`-fuzztime=1m` per target)                                                                     | 🔴 `TODO` | Med    | 30min  | `FuzzWriteEvent`/`FuzzParseEventID` only run seed corpus in normal `go test` |

## Low impact — doc accuracy and test modernization

| Task                                                                                     | Status    | Impact | Effort | Evidence                                                                             |
| ---------------------------------------------------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------------------------------------ |
| Fix stale `defer stream.Close()` in source doc comments (3 occurrences)                  | 🔴 `TODO` | Low    | 10min  | `doc.go:19`, `stream.go:22`, `stream.go:72` — should show `io.Closer` error handling |
| Fix stale `defer stream.Close()` in README examples (4 occurrences)                      | 🔴 `TODO` | Low    | 10min  | `README.md:49,74,86,149` — users copy-pasting get `errcheck` lint failures           |
| Modernize `context.WithCancel` to `t.Context()` in `stream_test.go` (5 occurrences)      | 🔴 `TODO` | Low    | 20min  | `stream_test.go:119,139,169,286,450` — Go 1.24+ testing idiom                        |
| Modernize `go func()` to `wg.Go` in tests where a WaitGroup is in scope (12 occurrences) | 🔴 `TODO` | Low    | 30min  | `stream_test.go` (8), `broadcaster_test.go` (2), `integration_test.go` (1)           |
