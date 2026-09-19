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

| Status    | Item                                                                                           | Notes                                                                                                                                                                                                                                                                                                                              |
| --------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Remove the now-inert `GOEXPERIMENT=jsonv2` exports (devShell, flake apps, CI, scripts, .envrc) | Verified unnecessary under Go 1.27 (2026-09-19, both modules build/test without it). Kept during the 1.26→1.27 transition as belt-and-braces; delete once no consumer machine runs a 1.26 toolchain against this repo. Low priority, wide-but-trivial diff.                                                                        |
| 🔴 `TODO` | Watch the first scheduled `datastar-compat.yml` run (Monday 2026-09-21 05:00 UTC)              | The workflow was dry-run verified locally 2026-09-19 (clone → bump to latest go-sse/ssetest → both go-datastar modules green), but the first real scheduled run proves the runner environment (toolchain via `go-version-file`, proxy access).                                                                                     |
| 🔴 `TODO` | Post-v0.6.1 companion work: bump go-datastar's go-sse/ssetest pins and tag datastartest | Pairing rule (CONTRIBUTING step 9): go-datastar still pins go-sse v0.6.0 / ssetest v0.3.0 while master is v0.6.1. Its `go` directives were separately bumped to 1.27.1 on 2026-09-19. Source: [09-19 report §f9](docs/status/2026-09-19_22-44_todo-backlog-execution-pass.md).                                              |
| 🔴 `TODO` | Enable GitHub private vulnerability reporting (repo setting)                       | SECURITY.md advertises the channel, but the setting is unset; the REST API rejects it (`422` — `private_vulnerability_reporting` is not part of `security_and_analysis`). ~30s in Settings → Code security and analysis. Source: report §f13.                                                                                      |

## Harvested backlog (2026-09-19 report §f, P2)

Every item below was verified still-open against the code on 2026-09-19
before being added (grep evidence noted where it matters).

| #   | Item                                                       | Notes                                                                                          |
| --- | ---------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| 14  | CI `coverage-gate` job                                     | The 90%/95% thresholds are local-only (`nix run .#coverage-gate`); ci.yml has no gate step.    |
| 15  | shellcheck CI job for `scripts/*.sh`                       | The actionlint workflow shellchecks `run:` blocks only; repo scripts are uncovered.            |
| 16  | shfmt (or a treefmt sh formatter)                          | treefmt formats Go + Nix only; `scripts/*.sh` are not format-gated.                            |
| 17  | templ CLI pin (`@v0.3.1020`) refresh policy                | Dependabot cannot bump `go run @version` pins; add a CONTRIBUTING note or a drift check.       |
| 18  | Benchmark regression tracking (benchstat vs a baseline)    | Perf claims in FEATURES.md are hand-pinned measurements.                                       |
| 19  | Godoc examples for `SendLines`/`SendKeyed`                 | Only `KeyedLines` has one today.                                                               |
| 20  | README: link the four guides                               | README.md has zero `docs/guides` references.                                                   |
| 21  | Document `sse.write_short` where error codes are listed    | AGENTS.md conventions list and README do not mention it.                                       |
| 22  | Direct unit tests for `forEachLine`                        | Covered only transitively via `splitLines` + the conformance corpora.                          |
| 23  | Unit tests for the three examples' `listenAddr()` override | No tests exist for the PORT env helpers.                                                       |
| 24  | Pin golangci-lint in flake.nix                             | Nixpkgs floats; the verify.sh pin-check flags skew only after it bit.                          |
| 25  | ssetest coverage 98.4% → higher                            | Remaining lines are likely `collect.go` error paths.                                           |
| 27  | `nix run .#smoke` flake app                                | Wrap `scripts/smoke-examples.sh` for symmetry with the other apps.                             |
| 28  | Re-check example docs for stale absolute port URLs         | After the PORT override landed, hardcoded `:8080`/`:8765`/`:8766` prose may lie.               |
| 29  | `Stream.Send` doc: mention the short-write error path      | The doc currently says only "write fails".                                                     |
| 30  | Link `docs/guides/` from AGENTS.md's docs pointer          | Four guides exist; AGENTS.md does not reference them.                                          |

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
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the browser-E2E scope decision first — see [docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md](docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md) (Option B vs C). The SUPERB plan's default (stay blocked) was applied 2026-08-29; the real DataStar JS client remains manually verified against the example server ([2026-08-05 archived report](docs/status/archived/2026-08-05_10-15_datastar-example-cdn-url-fix.md)). |

## Declined

| Status    | Item                                             | Reason                                                                                                                                                                                                                                                                                                                                        |
| --------- | ------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ⚪ `WONT` | Rewrite the misleading auto-git commit `38e79aa` | SUPERB plan D2 default applied 2026-08-29: amending a month-old published commit violates the no-history-rewrite rule for a cosmetic gain. The AGENTS.md auto-git Gotcha documents the misleading message ("expand test coverage" describes a coverage-reducing deletion). Revisit only on an explicit user order — amend + force-with-lease. |

## Resolved by external change (was open 2026-09-03)

| Former item                                                        | Resolution                                                                                                                                                                                                                             |
| ------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CI: treefmt/format gate job (16-36 §f11)                           | Already existed: `checks.format` in `flake.nix` runs inside `nix flake check` (the CI `nix` job). It genuinely bites — it failed this session's first `nix flake check` on an unformatted staged file. The report's premise was stale. |
| Dependabot/Renovate for the GitHub Actions SHA pins (16-36 §f30)   | `.github/dependabot.yml` configured 2026-09-15 (gomod ×2 + github-actions, weekly, grouped); update runs are green and PR #2 is open. The Node-20 deprecation warnings the item cited are addressed by those bumps.                    |
| gopls `stdversion` friction on `encoding/json/v2` (16-36 WONT row) | Disappeared with the `go 1.27.1` directive bump (2026-09-19) exactly as the WONT row predicted ("intrinsic until Go 1.27, disappears then"). Row retired.                                                                              |
