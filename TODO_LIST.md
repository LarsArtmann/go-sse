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

## Predicate filtering — correctness gaps

Open items identified in the predicate-filtering self-review and gap-closure
reports. All are bounded, code-level fixes.

| Status    | Item                                                                   | Notes                                                                                                                                                     |
| --------- | ---------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Add `TestIntegration_ReplayFiltered`                                   | HTTP round-trip with a `FilteredEventStore` implementation and a fallback store. Every other public API has an integration test; this is the asymmetry.   |
| 🔴 `TODO` | Add `TestSubscribeFilter_ShutdownDrainsFilteredSubscribers`            | Verify `Shutdown(ctx)` drain works correctly when subscribers have predicates. The drain path checks `len(sub.ch)` which should be unaffected, but untested. |
| 🔴 `TODO` | Strengthen race test assertion                                         | `TestSubscribeFilter_ConcurrentRace` checks `received.Load() > 0` — change to a meaningful threshold (e.g. `>= 500`) accounting for non-blocking drops.  |
| 🔴 `TODO` | Document panic contract on `ReplayFiltered`                            | `broadcaster.go` documents "panics crash by design" for `SubscribeFilter`. `ReplayFiltered` in `replay.go` has the same risk but the contract is not documented there. |

## Production readiness

| Status    | Item                                     | Notes                                                                                                                                                                                |
| --------- | ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 🔴 `TODO` | Scale profile: 64-buffer × N subscribers | Memory and latency characterization at subscriber counts (100/1k/10k). Produces a report, not a code change. Informs whether the buffer size or backpressure policy needs to change. |

## DataStar integration — follow-up items

The core DataStar support (`KeyedLines`, `SendLines`, `WriteKeyedLines`,
`SendKeyed`) is shipped. These items are lower-priority follow-ups:

| Status       | Item                                                        | Notes                                                                           |
| ------------ | ----------------------------------------------------------- | ------------------------------------------------------------------------------- |
| 🔴 `TODO`    | Point a real DataStar JS client at a go-sse example server  | Ultimate integration verification — confirm events parse correctly in a browser |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the real-client verification first                                     |

## Release

| Status    | Item               | Notes                                                                                                                    |
| --------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------ |
| 🔴 `TODO` | Cut v0.4.0 release | Tag, release notes, GitHub release. v0.3.0 was tagged without code changes; v0.4.0 will carry DataStar + lifecycle + predicate filtering. |

Completed work lives in [`CHANGELOG.md`](CHANGELOG.md), not here.
