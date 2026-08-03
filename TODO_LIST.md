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

## Verification & correctness

| Status    | Item                                                       | Notes                                                                                                                                                     |
| --------- | ---------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Point a real DataStar JS client at a go-sse example server | Ultimate integration verification — confirm events parse correctly in a browser now that `data-style:width` is correct. Unblocks the CI browser test below. |

## Blocked

| Status       | Item                                                        | Blocker                                           |
| ------------ | ----------------------------------------------------------- | ------------------------------------------------- |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the real-client verification above first |
