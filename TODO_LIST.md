# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, see [ROADMAP.md](ROADMAP.md).
> Items are ranked by Pareto impact. Status is verified, not assumed.
> Completed work lives in [`CHANGELOG.md`](CHANGELOG.md), not here.

State after the 2026-09-19 execution pass: the entire 2026-09-03 harvested
backlog is closed (details in the CHANGELOG `[Unreleased]` section) — the
correctness tests, the CI/tooling hardening, the docs (SECURITY.md, guides,
godoc examples), and the go-datastar cross-repo job. The pass also unblocked
a red master (the `ssetest/go.mod` half of the Go 1.27 bump was staged but
uncommitted) and fixed a real `WriteEvent` defect it found on the way
(short writes were silently truncated; they now surface `io.ErrShortWrite`).
Two findings resolved stale items without work: the CI format gate already
existed (`checks.format` inside `nix flake check` — it bit during this very
session), and Dependabot had been configured on 2026-09-15 with PRs flowing.

## Status legend

| Status           | Meaning                                                                                       |
| ---------------- | --------------------------------------------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                                                                     |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                                                     |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed.                                       |
| ⚪ `WONT`        | Deliberately declined, with the reason inline. Revisit only if the trigger condition changes. |

## Open items

| Status    | Item                                                                                            | Notes                                                                                                                                                                                                                                                              |
| --------- | ----------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Merge the two open automated PRs after master is green again                                     | #1 `chore(nix): weekly flake update 2026-09-07` (in-workflow gate was green; superseded runs auto-close only when a NEW drift appears) and #2 `chore(deps): bump the actions group` (its CI failed only because of the since-fixed ssetest go-directive red; Dependabot rebases on master pushes). Both need a human review-merge. |
| 🔴 `TODO` | Remove the now-inert `GOEXPERIMENT=jsonv2` exports (devShell, flake apps, CI, scripts, .envrc)   | Verified unnecessary under Go 1.27 (2026-09-19, both modules build/test without it). Kept during the 1.26→1.27 transition as belt-and-braces; delete once no consumer machine runs a 1.26 toolchain against this repo. Low priority, wide-but-trivial diff. |
| 🔴 `TODO` | Watch the first scheduled `datastar-compat.yml` run (Monday 2026-09-21 05:00 UTC)                | The workflow was dry-run verified locally 2026-09-19 (clone → bump to latest go-sse/ssetest → both go-datastar modules green), but the first real scheduled run proves the runner environment (toolchain via `go-version-file`, proxy access). |

## CI & tooling

| Status    | Item                                                          | Notes                                                                                                                                                                                                                                                                |
| --------- | ------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ⚪ `WONT` | git pre-push hook calling `scripts/verify.sh --fast`          | Deliberately declined for now: the auto-git daemon bypasses manual git flow entirely, so hooks gate neither its commits nor the user's real workflow; `scripts/verify.sh` + CI are the effective gates (same reasoning as the 07-27 report's pre-commit WONT, extended to pre-push). Revisit if pushes start shipping red. |
| ⚪ `WONT` | `testing/synctest` for the `CollectWithTimeout` tests         | The `testing/synctest` guidelines prohibit network I/O inside a bubble, and every `Collect*` helper owns a real-socket `httptest` server. A fake-net rewrite would test a different transport than production. Source: 2026-08-29 execution pass.                     |
| ⚪ `WONT` | Remaining gopls "unnecessary type argument" infos             | All inferable type arguments in tests were removed (2026-08-29). What remains is explicit by necessity: `NewBroadcaster[T]()` has no args to infer from and `WithBufferSize[T](…)`'s T appears only in its return type. `gopls check` CLI reports 0 diagnostics; the rest are editor-only hints. |

## Cross-repo

| Status    | Item                                                                | Notes                                                                                                                                                                                                                                        |
| --------- | ------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ⚪ `WONT` | Drop `replace` directives from the tagged `datastartest/go.mod` now | Inert for consumers (dependency replaces are ignored), so there is nothing to fix until the next datastartest tag — then drop them first (go-datastar's own checklist prescribes this). Source: [18-25 report a6](docs/status/archived/2026-08-29_18-25_superb-plan-execution-releases-and-ci-hardening.md). |

## Blocked

| Status       | Item                                                        | Blocker                                                                                                                                                                                                                                                                                                                                                             |
| ------------ | ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the browser-E2E scope decision first — see [docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md](docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md) (Option B vs C). The SUPERB plan's default (stay blocked) was applied 2026-08-29; the real DataStar JS client remains manually verified against the example server ([2026-08-05 archived report](docs/status/archived/2026-08-05_10-15_datastar-example-cdn-url-fix.md)). |

## Declined

| Status    | Item                                             | Reason                                                                                                                                                                                                                                                              |
| --------- | ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ⚪ `WONT` | Rewrite the misleading auto-git commit `38e79aa` | SUPERB plan D2 default applied 2026-08-29: amending a month-old published commit violates the no-history-rewrite rule for a cosmetic gain. The AGENTS.md auto-git Gotcha documents the misleading message ("expand test coverage" describes a coverage-reducing deletion). Revisit only on an explicit user order — amend + force-with-lease. |

## Resolved by external change (was open 2026-09-03)

| Former item                                                            | Resolution                                                                                                                                                                                                                     |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CI: treefmt/format gate job (16-36 §f11)                               | Already existed: `checks.format` in `flake.nix` runs inside `nix flake check` (the CI `nix` job). It genuinely bites — it failed this session's first `nix flake check` on an unformatted staged file. The report's premise was stale. |
| Dependabot/Renovate for the GitHub Actions SHA pins (16-36 §f30)       | `.github/dependabot.yml` configured 2026-09-15 (gomod ×2 + github-actions, weekly, grouped); update runs are green and PR #2 is open. The Node-20 deprecation warnings the item cited are addressed by those bumps. |
| gopls `stdversion` friction on `encoding/json/v2` (16-36 WONT row)     | Disappeared with the `go 1.27.1` directive bump (2026-09-19) exactly as the WONT row predicted ("intrinsic until Go 1.27, disappears then"). Row retired.                                                                       |
