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

### Production readiness — extracted from ROADMAP section 1

Bounded items promoted out of the production-readiness theme. The remaining
explorations (backpressure policy, observability shape) stay in ROADMAP until
they narrow to a single approach.

| Status    | Item                                     | Notes                                                                                                                                                                                |
| --------- | ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 🟢 `DONE` | Graceful-shutdown helper                 | Shipped as `Broadcaster.Shutdown(ctx) error` (drain respecting context deadline) plus `Broadcaster.Health() BroadcasterHealth` for health checks. Signal handling stays in the consumer. |
| 🟢 `DONE` | Configurable subscriber buffer size      | Shipped as `WithBufferSize[T](size int) Option[T]` applied at construction via `NewBroadcaster[T](opts ...)`. Default of 64 preserved; non-positive values fall back to the default.   |
| 🔴 `TODO` | Scale profile: 64-buffer × N subscribers | Memory and latency characterization at subscriber counts (100/1k/10k). Produces a report, not a code change. Informs whether the buffer size or backpressure policy needs to change. |

### DataStar integration — follow-up items

The core DataStar support (`KeyedLines`, `SendLines`, `WriteKeyedLines`,
`SendKeyed`, `JSONSignals`) is shipped. These items are lower-priority
follow-ups:

| Status       | Item                                                        | Notes                                                                           |
| ------------ | ----------------------------------------------------------- | ------------------------------------------------------------------------------- |
| 🔴 `TODO`    | Point a real DataStar JS client at a go-sse example server  | Ultimate integration verification — confirm events parse correctly in a browser |
| 🔴 `TODO`    | Cut 0.3.0 release                                           | Tag, release notes, GitHub release. New API = minor bump                        |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the real-client verification first                                     |

Completed work lives in [`CHANGELOG.md`](CHANGELOG.md), not here.
