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

Follow-up pass (2026-09-19, later session): the verify gate was re-run to a
green `ALL CHECKS PASSED`; PR #1 was closed as superseded (master's
flake.lock was already newer than the PR's 09-07 bump, so the branch
resolved to a zero diff); PR #2 (dependabot actions bumps) was rebased and
squash-merged with every check green; v0.6.1 was confirmed cut and
published; and a docs-health HARVEST pass pulled the execution-pass
report's §f backlog into the tables below, every item verified against the
code before adding.

## Status legend

| Status           | Meaning                                                                                       |
| ---------------- | --------------------------------------------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                                                                     |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                                                     |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed.                                       |
| ⚪ `WONT`        | Deliberately declined, with the reason inline. Revisit only if the trigger condition changes. |

## Open items

| Status    | Item                                                                                           | Notes                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| --------- | ---------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Watch the next scheduled `datastar-compat.yml` run (Mondays 05:00 UTC)                         | Bump MECHANICS verified against the sseparse-carrying release 2026-09-29: exact workflow commands on a go-datastar copy, proxy-only — `go get ssetest@v0.4.0` auto-added `require sseparse v0.1.0 // indirect` to datastartest and both go-datastar modules test green, so no pin-bump-set extension is needed. Remaining unknown is only the live runner environment (toolchain via `go-version-file`, proxy access). go-datastar master now carries the v0.6.1 pins already, so the next run should report no-op bumps. |

## Harvested backlog (2026-09-19 report §f, P2)

Every item below was verified still-open against the code on 2026-09-19
before being added (grep evidence noted where it matters).

| #  | Item                                                       | Notes                                                                                                                                                                                                                                                                                                                   |
| -- | ---------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 24 | Pin golangci-lint in flake.nix                             | Nixpkgs floats; the verify.sh pin-check flags skew only after it bit.                                                                                                                                                                                                                                                   |

## Harvested backlog (2026-09-29 07-03 report §f, post-release)

Harvested 2026-09-29 from
[docs/status/2026-09-29_07-03_ci-health-vendorhash-repair-pr-closure.md](docs/status/2026-09-29_07-03_ci-health-vendorhash-repair-pr-closure.md)
§f; every row was verified still-open against the code at harvest time
(§f items already closed by commits were deleted, not carried).
Low/Roadmap-fuel items (§f 26–50) stay in the source report, summarized
in [ROADMAP.md](ROADMAP.md).

| Status    | Item                                                                | Notes                                                                                                                                                                                                                                                                                               |
| --------- | ------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Raise coverage-gate floors to current levels                        | The `coverage-gate` flake app still enforces 90/95/95 while actual coverage is library 99.3 / ssetest 100.0 / sseparse 99.2 (07-03 report cover). ssetest 95→100 is unambiguous; consider 99 for library and sseparse. Source: §f8.                                                                   |
| 🔴 `TODO` | Verify tag-CI ran green for `sseparse/v0.1.0` + `ssetest/v0.4.0`    | Same frozen-tag-CI class as go-datastar v0.6.1: the workflow file at a tag is immutable, so red or never-triggered tag-CI is permanent. Check the two tag runs; if red, decide on a re-cut. Source: §f12.                                                                                             |
| 🔴 `TODO` | Evaluate go-error-family v0.11.0 for the root module                | go-datastar already bumped; a dependabot PR will arrive anyway, so get ahead of it and do the vendor-hash dance once. Source: §f14.                                                                                                                                                                   |
| 🔴 `TODO` | Recheck AGENTS.md directive claims on the next root tag             | Master directives are aligned at `go 1.27.1` ×3, but the released root tag can drift from master; re-verify the directive notes when the next root tag ships. Source: §f15.                                                                                                                          |
| 🔴 `TODO` | Local 5-minute fuzz soak on the bumped tree                         | Dependencies changed 2026-09-29 (go-sse v0.6.1, error-family v0.10.2, directive floor); CI's scheduled Fuzz runs are green, but a fresh local soak is cheap. Source: §f17.                                                                                                                            |
| 🔴 `TODO` | Annotate + archive the 01-58 sseparse-release status report         | Resolutions are known (tags live, consumers cascaded); annotate inline per docs-health ANNOTATE, then `git mv` to `docs/status/archived/` once fully resolved. Source: §f19.                                                                                                                         |
| 🔴 `TODO` | Verify coverage-gate measures the root as package `.`               | The documented scope rule says package `.`; if the app actually passes `./...` the root gate understates coverage by a wide margin. Source: §f21; app lives in flake.nix `coverage-gate`.                                                                                                              |

## CI & tooling

| Status    | Item                                                  | Notes                                                                                                                                                                                                                                                                                                                      |
| --------- | ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ⚪ `WONT` | git pre-push hook calling `scripts/verify.sh --fast`  | Deliberately declined for now: the auto-git daemon bypasses manual git flow entirely, so hooks gate neither its commits nor the user's real workflow; `scripts/verify.sh` + CI are the effective gates (same reasoning as the 07-27 report's pre-commit WONT, extended to pre-push). Revisit if pushes start shipping red. |
| ⚪ `WONT` | `testing/synctest` for the `CollectWithTimeout` tests | The `testing/synctest` guidelines prohibit network I/O inside a bubble, and every `Collect*` helper owns a real-socket `httptest` server. A fake-net rewrite would test a different transport than production. Source: 2026-08-29 execution pass.                                                                          |
| ⚪ `WONT` | Remaining gopls "unnecessary type argument" infos     | All inferable type arguments in tests were removed (2026-08-29). What remains is explicit by necessity: `NewBroadcaster[T]()` has no args to infer from and `WithBufferSize[T](…)`'s T appears only in its return type. `gopls check` CLI reports 0 diagnostics; the rest are editor-only hints.                           |

## Cross-repo

| Status    | Item                                                                | Notes                                                                                                                                                                                                                                                                                                        |
| --------- | ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| ⚪ `WONT` | Drop `replace` directives from the tagged `datastartest/go.mod` now | Inert for consumers (dependency replaces are ignored), so there is nothing to fix until the next datastartest tag — then drop them first (go-datastar's own checklist prescribes this). Source: [18-25 report a6](docs/status/archived/2026-08-29_18-25_superb-plan-execution-releases-and-ci-hardening.md). |

## Blocked

| Status       | Item                                                        | Blocker                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| ------------ | ----------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔵 `BLOCKED` | go-datastar v0.6.2: re-cut or accept frozen red tag-CI?     | User decision (07-03 report §g1): the four v0.6.1 tags carry permanently red tag-CI (immutable old workflow pin + stale hash at the tag). If yes, execution is ~30 min; no technical blocker — release content was verified green before tagging.                                                                                                                                       |
| 🔵 `BLOCKED` | Workflow-approval gate for Actions-created PRs              | GitHub holds ALL pull_request CI for GITHUB_TOKEN-authored PRs in `action_required` until a human approves (PR #4 sat unseen for 26h). Options: approve-bot, workflow_run pattern, or repo setting; needs one policy decision covering go-sse and go-datastar. Source: 07-03 report §c1/§f2.                                                                                            |
| 🔵 `BLOCKED` | buildflow normalize: derive go directive from max(dep directives) | The normalize step hardcodes its target directive and lowered go-sse modules to `go 1.27` twice on 2026-09-29, fighting the hard `1.27.1` floor that go-sse v0.6.1 on the proxy imposes. The fix belongs in buildflow (fleet-wide), not this repo; the flip-flop is currently held back only by the AGENTS.md rationale. Source: 07-03 report §c3/§f3.                                    |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the browser-E2E scope decision first — see [docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md](docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md) (Option B vs C). The SUPERB plan's default (stay blocked) was applied 2026-08-29; the real DataStar JS client remains manually verified against the example server ([2026-08-05 archived report](docs/status/archived/2026-08-05_10-15_datastar-example-cdn-url-fix.md)). |

## Declined

| Status    | Item                                             | Reason                                                                                                                                                                                                                                                                                                                                        |
| --------- | ------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ⚪ `WONT` | Rewrite the misleading auto-git commit `38e79aa` | SUPERB plan D2 default applied 2026-08-29: amending a month-old published commit violates the no-history-rewrite rule for a cosmetic gain. The AGENTS.md auto-git Gotcha documents the misleading message ("expand test coverage" describes a coverage-reducing deletion). Revisit only on an explicit user order — amend + force-with-lease. |

## Resolved by external change (was open 2026-09-03)

| Former item                                                                      | Resolution                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| -------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CI: treefmt/format gate job (16-36 §f11)                                         | Already existed: `checks.format` in `flake.nix` runs inside `nix flake check` (the CI `nix` job). It genuinely bites — it failed this session's first `nix flake check` on an unformatted staged file. The report's premise was stale.                                                                                                                                                                                                        |
| Dependabot/Renovate for the GitHub Actions SHA pins (16-36 §f30)                 | `.github/dependabot.yml` configured 2026-09-15 (gomod ×2 + github-actions, weekly, grouped); update runs are green and PR #2 is open. The Node-20 deprecation warnings the item cited are addressed by those bumps.                                                                                                                                                                                                                           |
| gopls `stdversion` friction on `encoding/json/v2` (16-36 WONT row)               | Disappeared with the `go 1.27.1` directive bump (2026-09-19) exactly as the WONT row predicted ("intrinsic until Go 1.27, disappears then"). Row retired.                                                                                                                                                                                                                                                                                     |
| Release the sseparse/ssetest split (open from the 2026-09-29 go-daemon feedback) | Executed 2026-09-29 02:22 (commit `7cd5b3a`): tags `sseparse/v0.1.0` + `ssetest/v0.4.0` exist, ssetest's `replace` dropped (it requires `sseparse v0.1.0` as a tagged module), `release-verify.sh` gained the `sseparse/*` case in the same commit. Both tags re-verified post-push 2026-09-29 (proxy index, zip, from-scratch consumer build+import — both `RELEASE PROBE PASSED`). Root library unchanged — no root tag needed, as planned. |
