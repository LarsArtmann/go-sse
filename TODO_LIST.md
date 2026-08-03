# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, see [ROADMAP.md](ROADMAP.md).
> Items are ranked by Pareto impact. Status is verified, not assumed.
> Completed work lives in [`CHANGELOG.md`](CHANGELOG.md), not here.

## Status legend

| Status           | Meaning                                                 |
| ---------------- | ------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                               |
| 🟡 `IN_PROGRESS` | Actively being worked on.                               |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed. |

## Production readiness

| Status    | Item                                     | Notes                                                                                                                                                                                |
| --------- | ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 🔴 `TODO` | Scale profile: 64-buffer × N subscribers | Memory and latency characterization at subscriber counts (100/1k/10k). Produces a report, not a code change. Informs whether the buffer size or backpressure policy needs to change. |

## Verification & correctness

| Status    | Item                                                               | Notes                                                                                                                                                                |
| --------- | ------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Verify `data-bind:style` attribute in `example/datastar/main.go`   | The progress bar uses `data-bind:style` (line 150) which was never verified against DataStar v1.0.2 client docs. May need `data-attr:style` or a different approach. |
| 🔴 `TODO` | Point a real DataStar JS client at a go-sse example server         | Ultimate integration verification — confirm events parse correctly in a browser. Unblocks the CI headless browser test below.                                        |
| 🔴 `TODO` | Verify internal markdown links resolve after planning-doc archival | Two planning docs moved to `docs/planning/archived/` — inbound links from status reports or living docs may be broken. Run `grep -roE '\]\([^)]+\)' *.md docs/`.     |

## Test quality

These items were flagged across multiple self-review sessions as correctness
gaps in the predicate-filtering test surface. All are small (S effort) and
bounded.

| Status    | Item                                                                                 | Notes                                                                                            |
| --------- | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| 🔴 `TODO` | `TestSubscribeFilter_DropPolicyRespected` — full buffer drops matching events        | Verifies that a filtered subscriber with a full buffer drops matching (not non-matching) events. |
| 🔴 `TODO` | `TestSubscribeFilter_BroadcastManyMixedSubscribers` — half filtered, half unfiltered | Verifies correct partition when half the subscribers have predicates and half don't.             |
| 🔴 `TODO` | `ExampleReplayFiltered` in `example_test.go`                                         | Every other public API has a godoc example; ReplayFiltered is the asymmetry.                     |

## Blocked

| Status       | Item                                                        | Blocker                                           |
| ------------ | ----------------------------------------------------------- | ------------------------------------------------- |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the real-client verification above first |
