# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, see [ROADMAP.md](ROADMAP.md).
> Items are ranked by Pareto impact. Status is verified, not assumed.
> Completed work lives in [`CHANGELOG.md`](CHANGELOG.md), not here.

Every item from the 2026-08-29 full-list execution pass is either done
(→ CHANGELOG `[Unreleased]`), dispositioned below, or still blocked. The list
is intentionally short: correctness, fuzz depth, CI gates, and the docs batch
all shipped in that pass.

## Status legend

| Status           | Meaning                                                                                       |
| ---------------- | --------------------------------------------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                                                                     |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                                                     |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed.                                       |
| ⚪ `WONT`        | Deliberately declined, with the reason inline. Revisit only if the trigger condition changes. |

## Correctness & safety

Nothing open — the `safeDropCall` batch, `OnDrop(nil)` tests, and the
`eventBrand.Name()` test shipped in the 2026-08-29 pass (root coverage 99.3%).

## Parser parity & fuzz depth

| Status    | Item                                                  | Notes                                                                                                                                                                                                                                                                                            |
| --------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| ⚪ `WONT` | `testing/synctest` for the `CollectWithTimeout` tests | The `testing/synctest` guidelines prohibit network I/O inside a bubble, and every `Collect*` helper owns a real-socket `httptest` server. A fake-net rewrite would test a different transport than production. The cost is one ~200 ms test; keep it real. Source: 2026-08-29 execution pass.    |
| ⚪ `WONT` | Remaining gopls "unnecessary type argument" infos     | All inferable type arguments in tests were removed (2026-08-29). What remains is explicit by necessity: `NewBroadcaster[T]()` has no args to infer from and `WithBufferSize[T](…)`'s T appears only in its return type. `gopls check` CLI reports 0 diagnostics; the rest are editor-only hints. |
| ⚪ `WONT` | gopls `stdversion` friction on `encoding/json/v2`     | Root-caused: `jsonv2` std API is gated to a `go 1.27` directive in std metadata; the directive cannot exceed the 1.26.7 toolchain. The diagnostic is intrinsic until Go 1.27 and disappears then. Not a config problem; golangci-lint is clean.                                                  |

## CI & tooling

| Status    | Item                                                             | Notes                                                                                                                                                                                                                                                 |
| --------- | ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Schedule a periodic `nix flake update` + full-gate CI run (cron) | `.github/workflows/ci.yml` has no `schedule:` trigger, so dependency/input drift is only caught when someone bumps manually. A weekly cron running `nix flake update` + `nix flake check` (PR on change) would catch drift early. Source: [2026-07-27 report](docs/status/2026-07-27_10-26_removed-brand-name-test-and-self-review.md) item 50. |

Shipped in the 2026-08-29 pass: `scripts/verify.sh`, the examples+templ CI
job, the `nix flake check` CI job, the extended fuzz job (6 targets), the
pinned `govulncheck@v1.7.0`, and the ssetest coverage threshold (95%).

## Docs

| Status    | Item                                                                                  | Notes                                                                                                                                                                                                                                                                                      |
| --------- | ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 🔴 `TODO` | Document an auto-git commit-message review rule in AGENTS.md                          | Auto-git messages can be plausible-but-false for deletions: commit `38e79aa` says "expand test coverage" for a deletion that reduced it. Add a Gotcha telling sessions to sanity-check daemon-authored messages after the fact. Source: [2026-07-27 report](docs/status/2026-07-27_10-26_removed-brand-name-test-and-self-review.md) IMP4/item 25. |
| 🔴 `TODO` | Add `docs/status/AGENTS.md` report conventions with a mandatory coverage-delta field  | No status-report conventions file exists, so the 2026-07-27 coverage regression (100% → 99.5%) shipped unnoticed — coverage was never a report field. One page: report sections plus a mandatory `-cover` delta line. Source: [2026-07-27 report](docs/status/2026-07-27_10-26_removed-brand-name-test-and-self-review.md) item 46.        |

Shipped in the 2026-08-29 pass: `docs/guides/reconnection-and-retry.md` and
the CONTRIBUTING release checklist.

## Cross-repo

| Status    | Item                                                          | Notes                                                                                                                                                                                                                                      |
| --------- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 🔴 `TODO` | Writer goldens for DataStar patches in go-datastar core       | The datastartest-side parity batch shipped 2026-08-29 (fuzz corpus port, conformance seeds, README section); the writer-golden pinning belongs to go-datastar's core package and needs that repo's own conventions — not done in the pass. |
| 🔴 `TODO` | Commit + release the datastartest parity batch in go-datastar | Changes sit uncommitted in `/home/lars/projects/go-datastar` (`datastartest/README.md`, `reader_fuzz_test.go`, `testdata/fuzz/FuzzReadEvents/` ×51). Release a datastartest tag after the next ssetest release so both ship together.      |

## Blocked

| Status       | Item                                                        | Blocker                                                                                                                                                                                                                                                                                                                                                                           |
| ------------ | ----------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the browser-E2E scope decision first — see [docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md](docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md) (Option B vs C). The real DataStar JS client was manually verified against the example server on 2026-08-05 (`docs/status/archived/2026-08-05_10-15_datastar-example-cdn-url-fix.md`). |
| 🔵 `BLOCKED` | Correct or accept the misleading auto-git commit `38e79aa`  | User decision on history rewrite: amending a month-old published commit conflicts with the no-rewrite rule, but its message ("expand test coverage for Event type") describes a coverage-REDUCING deletion and actively misleads `git log` readers. Cheap compromise if left as-is: the AGENTS.md auto-git note above. Source: [2026-07-27 report](docs/status/2026-07-27_10-26_removed-brand-name-test-and-self-review.md) §g2/item 4. |
