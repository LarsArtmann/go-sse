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

Completed work lives in [`CHANGELOG.md`](CHANGELOG.md), not here.
