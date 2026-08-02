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

## Backlog

### DataStar integration — follow-up items

The core DataStar support (`KeyedLines`, `SendLines`, `WriteKeyedLines`,
`SendKeyed`, `JSONSignals`) is shipped. These items are lower-priority
follow-ups:

| Status | Item | Notes |
| ------ | ---- | ----- |
| 🔴 `TODO` | Point a real DataStar JS client at a go-sse example server | Ultimate integration verification — confirm events parse correctly in a browser |
| 🔴 `TODO` | Cut 0.3.0 release | Tag, release notes, GitHub release. New API = minor bump |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the real-client verification first |

### Previously shipped

All prior backlog items are complete and shipped under their respective
`CHANGELOG.md` entries:

- `KeyedLines` + `SendLines` + `WriteKeyedLines` + `SendKeyed` + `JSONSignals` — DataStar keyed-data-line support (0.3.0 `[Unreleased]`)
- `FuzzKeyedLines`, DataStar wire-format integration test, `BenchmarkKeyedLines`
- Flaky-test elimination (`TestIntegration_BroadcasterFanOut`, `TestStream_Heartbeat`) via deterministic channel sync
- Heartbeat comment-frame delivery and `Last-Event-ID` reconnection replay integration tests (real HTTP round-trips)
- Coverage raised to ~100% (incl. `Name()`, `MustParseEventID` success path, `splitLines` dead-branch removal, `Heartbeat` write-error path)
- `govulncheck` and `fuzz` CI jobs
- Doc/source/README examples fixed to `defer func() { _ = stream.Close() }()`
- Test modernization: `context.WithCancel(t.Context())` and `wg.Go`
