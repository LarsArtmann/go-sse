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

All previously-tracked items are complete and shipped under `[Unreleased]` in
`CHANGELOG.md`:

- Flaky-test elimination (`TestIntegration_BroadcasterFanOut`,
  `TestStream_Heartbeat`) via deterministic channel sync.
- Heartbeat comment-frame delivery and `Last-Event-ID` reconnection replay
  integration tests (real HTTP round-trips).
- Coverage raised to 100% (incl. `Name()`, `MustParseEventID` success path,
  `splitLines` dead-branch removal, `Heartbeat` write-error path).
- `govulncheck` and `fuzz` CI jobs.
- Doc/source/README examples fixed to `defer func() { _ = stream.Close() }()`.
- Test modernization: `context.WithCancel(t.Context())` and `wg.Go`.

Nothing actionable remains. Add new items here as they arise.
