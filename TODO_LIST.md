# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, see [ROADMAP.md](ROADMAP.md).
> Items are ranked by Pareto impact. Status is verified, not assumed.
> Completed work lives in [`CHANGELOG.md`](CHANGELOG.md), not here.

## Status legend

| Status           | Meaning                                                                                      |
| ---------------- | -------------------------------------------------------------------------------------------- |
| ✅ `DONE`        | Resolved. Removed from active TODO; lives in `CHANGELOG.md` and/or referenced status report. |
| 🔴 `TODO`        | Not started. Needs doing.                                                                    |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                                                    |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed.                                      |

## Verification & correctness

| Status    | Item                                                                   | Notes                                                                                                                                                                                                                                                                                                               |
| --------- | ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Port the ssetest fuzz regression seeds to `datastartest` (go-datastar) | ssetest now pins the trailing-LF regression (`testdata/fuzz/FuzzWriteReadRoundTrip/2ba7b6a0aaf94e65`) plus WPT seed vectors; `datastartest/testdata/fuzz/` is empty, so the same parser bug class would regress unnoticed there. The two parsers are deliberately duplicated and must stay in lockstep (AGENTS.md). |

> ✅ 2026-08-05 — Real DataStar JS client manually tested against `example/datastar` example server (browser session). Progress bar and status patches confirmed working end-to-end after the CDN URL fix (`docs/status/2026-08-05_10-15_datastar-example-cdn-url-fix.md`). Both `data-style:width` (data attribute) and the `[email protected]` placeholder bug are simultaneously verified.

## Blocked

| Status       | Item                                                        | Blocker                                           |
| ------------ | ----------------------------------------------------------- | ------------------------------------------------- |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the real-client verification above first |
