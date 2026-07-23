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

## Tier 1% — Foundation (delivers 51% of value)

| Task                                                                        | Status    | Impact   | Effort | Evidence                                                      |
| --------------------------------------------------------------------------- | --------- | -------- | ------ | ------------------------------------------------------------- |
| Create CI workflow (`.github/workflows/ci.yml`): test + lint + vet + race  | 🔴 `TODO` | Critical | 90min  | `.golangci.yml` configured but nothing runs it automatically  |
| LastEventIDFromRequest: validate untrusted header via ParseEventID          | 🔴 `TODO` | Critical | 45min  | `stream.go:201` — reads header with NewEventID (no validation) |

## Tier 4% — Quality & Correctness (delivers 64% of value)

| Task                                                                          | Status    | Impact | Effort | Evidence                                                |
| ----------------------------------------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------------- |
| Add fuzz tests for `WriteEvent` (serializer) and `ParseEventID` (validator)  | 🔴 `TODO` | High   | 100min | `event.go:96`, `event.go:44` — prime fuzz targets        |
| Add integration test with real `http.Server` (not just `httptest.NewRecorder`) | 🔴 `TODO` | High   | 60min  | All stream tests use `httptest.NewRecorder`             |
| Type safety: `Event.Retry` int → uint (make negative impossible)            | 🔴 `TODO` | High   | 45min  | `event.go:90` — allows negative retry (impossible state) |
| Type safety: `EventStore.EventsAfter(string)` → `EventsAfter(EventID)`      | 🔴 `TODO` | High   | 45min  | `replay.go:13` — loses branded type at boundary          |
| Test quality: replace `contains`/`itoa`/`errorResponseWriter` + modernize    | 🔴 `TODO` | Med    | 60min  | `stream_test.go:317`, `broadcaster_test.go:414`          |
| Heartbeat dedup: extract constant, use WriteHeartbeat in Stream.Heartbeat    | 🔴 `TODO` | Med    | 30min  | `event.go:142` and `stream.go:164` duplicate `": heartbeat\n\n"` |
| Edge-case tests: double-Close, BroadcastAfterClose, WriteRetry, unicode     | 🔴 `TODO` | Med    | 60min  | 0% coverage on `WriteRetry`; no double-Close test        |

## Tier 20% — Type Safety & API (delivers 80% of value)

| Task                                                                        | Status    | Impact | Effort | Evidence                                         |
| --------------------------------------------------------------------------- | --------- | ------ | ------ | ------------------------------------------------ |
| Stream.Close → io.Closer interface (return error)                          | 🔴 `TODO` | Med    | 30min  | `stream.go:120` — Close returns nothing, not error |
| Add `example/` directory with runnable server + client                     | 🔴 `TODO` | Med    | 45min  | No examples; README snippets are inline only      |

## Tier 80% — Release & Polish (delivers 20% of value)

| Task                                                                  | Status    | Impact | Effort | Evidence                                                       |
| --------------------------------------------------------------------- | --------- | ------ | ------ | -------------------------------------------------------------- |
| Create first git tag (v0.1.0) + pkg.go.dev link in README            | 🔴 `TODO` | Low    | 30min  | Zero tags exist; no pkg.go.dev reference                       |
| Fix ~11 imprecise file:line refs in FEATURES.md                      | 🔴 `TODO` | Low    | 45min  | Status report 21-23 found refs point to comments not decls     |
| Replace hardcoded test count in FEATURES.md with command reference   | 🔴 `TODO` | Low    | 5min   | FEATURES.md says "52 tests" — should point to command          |
| README: Event struct match source + drop policy implications section | 🔴 `TODO` | Low    | 30min  | README:112-122 simplified struct; no drop policy doc           |
| Verify SSE spec URL live + add build-tags note to CONTRIBUTING.md    | 🔴 `TODO` | Low    | 15min  | Never fetched spec URL; CONTRIBUTING omits build tags          |
