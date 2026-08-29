# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, see [ROADMAP.md](ROADMAP.md).
> Items are ranked by Pareto impact. Status is verified, not assumed.
> Completed work lives in [`CHANGELOG.md`](CHANGELOG.md), not here.

Every item from the 2026-08-29 full-list execution pass and its same-day
follow-up is either done (→ CHANGELOG `[Unreleased]` or git history),
dispositioned below, or still blocked. The list is intentionally short:
correctness, fuzz depth, CI gates, and the docs batch all shipped in that
pass; the cron workflow, both docs items, and the go-datastar writer goldens
shipped in the follow-up.

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

## Release

| Status    | Item                                                        | Notes                                                                                                                                                                 |
| --------- | ----------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Cut root `v0.6.0` + `ssetest v0.3.0` (full gate first)      | The 2026-08-29 value batch (18 Added + 7 Changed) is consumer-invisible until tagged. Follow the CONTRIBUTING release checklist exactly. The datastartest tag follows (pairing rule). Plan: [SUPERB pareto plan](docs/planning/2026-08-29_20-10_SUPERB-release-the-value-pareto-plan.md) P1–P3. |

## CI & tooling

| Status    | Item                                            | Notes                                                                                                                                                                                                                                       |
| --------- | ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Add `actionlint` to CI and/or the devShell      | The workflows are only YAML-parsed today. go-datastar already runs actionlint always-on in its CI (`5887043`); this repo's `flake-update.yml` was validated only with a borrowed devShell instance. Catches expression/step bugs YAML parsing cannot. |

| 🔴 `TODO` | Auto-close superseded `chore/flake-update-*` PRs in the workflow | Weekly runs pile up open PRs if ignored; close the previous update PR before opening a new one (`gh pr list --head`, keep newest). Plan ref: P8. |
| 🔴 `TODO` | Explicit shellcheck CI step for workflow run-blocks   | actionlint's shellcheck integration is silently optional — prove it or run shellcheck directly on the extracted run blocks. Plan ref: P9. |

The weekly `nix flake update` cron workflow shipped in the
2026-08-29 follow-up pass (`.github/workflows/flake-update.yml`: Mondays
04:00 UTC; the full `nix flake check` gate runs in-workflow on the bumped
inputs before the PR opens, because GITHUB_TOKEN-opened PRs never trigger
CI workflows).

Shipped in the 2026-08-29 pass: `scripts/verify.sh`, the examples+templ CI
job, the `nix flake check` CI job, the extended fuzz job (6 targets), the
pinned `govulncheck@v1.7.0`, and the ssetest coverage threshold (95%).

## Docs

| Status    | Item                                  | Notes                                                                                                                                                                                                                             |
| --------- | ------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Status-report conventions v1.1        | Allow an optional `## TL;DR` section, define cover-line scope for cross-repo sessions, note that the file specializes the global report format for this repo. Plan ref: P10.                                                        |
| 🔴 `TODO` | `docs/status/_template.md`            | Copy-pasteable report skeleton (title, preamble, `- cover:` line, a–g sections); link from the conventions file and the root AGENTS.md pointer. Plan ref: P11.                                                                       |

Shipped in the 2026-08-29 follow-up pass: the auto-git commit-message review
Gotcha (AGENTS.md) and `docs/status/AGENTS.md` report conventions with the
mandatory coverage-delta line.

Shipped in the 2026-08-29 pass: `docs/guides/reconnection-and-retry.md` and
the CONTRIBUTING release checklist.

## Cross-repo

Shipped in the 2026-08-29 follow-up pass: writer goldens for DataStar patches
in go-datastar core (`wire_golden_test.go` — 12 goldens pinning the exact
SSE wire bytes of every patch family, including data-line ordering, default
elision, and the multi-line `elements` splitting). The datastartest parity
batch itself was committed by the auto-git daemon as `d032dc5` — the old
row's "changes sit uncommitted" premise went stale.

| Status    | Item                                                        | Notes                                                                                                                                                                                                                                  |
| --------- | ----------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Land the go-datastar session batch behind its FULL gate     | Commit ONLY `wire_golden_test.go` + the CHANGELOG bullet (the tree is mixed with a concurrent session's work — coordinate first, never bulk-commit); then run that repo's full gate: workspace tests ×3 modules, `GOWORK=off` isolation ×3, erraudit ×3, `go work sync` idempotency. Plan ref: P5/P15. |

| Status       | Item                                      | Blocker                                                                                                                                                                  |
| ------------ | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 🔵 `BLOCKED` | Release the datastartest parity-batch tag in go-datastar | Cut after the next ssetest release so both ship together (pairing rule); tagging/publishing is user-gated. The batch itself is committed (`d032dc5`, `7fa8ed4`). |

## Blocked

| Status       | Item                                                        | Blocker                                                                                                                                                                                                                                                                                                                                                                           |
| ------------ | ----------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the browser-E2E scope decision first — see [docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md](docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md) (Option B vs C). The real DataStar JS client was manually verified against the example server on 2026-08-05 (`docs/status/archived/2026-08-05_10-15_datastar-example-cdn-url-fix.md`). |
| 🔵 `BLOCKED` | Correct or accept the misleading auto-git commit `38e79aa`  | User decision on history rewrite: amending a month-old published commit conflicts with the no-rewrite rule, but its message ("expand test coverage for Event type") describes a coverage-REDUCING deletion and actively misleads `git log` readers. Cheap compromise if left as-is: the AGENTS.md auto-git note above. Source: [2026-07-27 report](docs/status/2026-07-27_10-26_removed-brand-name-test-and-self-review.md) §g2/item 4. |
| 🔵 `BLOCKED` | Canonical CHANGELOG policy for CI-gate lines (D1) | User ruling needed: the Policy section says chore-tier CI wiring is git-history-only, but the same pass added CI-job lines under Added — the precedent followed for the cron-workflow line. Plan §5 D1. |
