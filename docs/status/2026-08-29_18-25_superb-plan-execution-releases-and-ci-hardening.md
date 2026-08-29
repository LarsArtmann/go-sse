# Status Report — 2026-08-29 18:25 — SUPERB plan execution: releases cut, CI hardened, gates resolved

Executed the full [SUPERB Pareto plan](../planning/2026-08-29_20-10_SUPERB-release-the-value-pareto-plan.md):
both go-sse releases cut and consumer-verified, the go-datastar batch verified
behind its full gate, CI red-master repaired and hardened, quick wins shipped,
and all three decision gates resolved via their documented defaults. Final
gate: `scripts/verify.sh` (full) ALL CHECKS PASSED, `nix flake check
--all-systems` green, CI green on the last pushed commit, actionlint workflow
green on first run, flake-update dry-run green.

- cover: library 99.3% (=), ssetest 97.2% (=)

## TL;DR

- **go-sse v0.6.0 and ssetest v0.3.0 are live** — tagged on CI-green commits, worktree-validated, proxy-serving, pkg.go.dev-indexed (root), GitHub-released, and proven by scratch consumers that `go get`, build, and run the real wire format.
- **Master CI was red on every push before this session** (fuzz property bug + lint version skew); both fixed, root-caused, and pinned against regression.
- **go-datastar is fully squared away**: writer goldens landed (`a0c0aea`), its complete gate re-run green (workspace, isolation ×3, erraudit ×3, sync idempotency), and `datastartest/v0.3.0` consumer-verified from the proxy.
- All TODO_LIST items closed except the deliberately blocked browser-E2E decision (D3).

## a) FULLY DONE

| # | Work | Evidence |
| --- | --- | --- |
| 1 | Fixed red master CI: `FuzzKeyedLines` asserted its prefix property for keys containing CR/LF (out of contract) | `bdd08c2`; property guarded in `fuzz_test.go`, CI crasher `key="\n"` pinned as `testdata/fuzz/FuzzKeyedLines/524783c5de59faaa`, 20s local fuzz run 2.6M execs PASS; CI run 33268470421 all-green |
| 2 | Fixed CI lint failure: `goconst` counting difference between golangci-lint 2.12 (CI) and 2.13 (flake) | `bdd08c2`; `"SSEEvent"` extracted to `eventBrandName` + test-local `wantBrandName` (independent oracle), CI pinned to v2.13.1 matching the flake; Lint job green since |
| 3 | P1 pre-release full gate | `scripts/verify.sh` (FULL incl. `nix flake check`) ALL CHECKS PASSED; `nix flake check --all-systems` all checks passed; coverage-gate 99.3%/97.2% OK |
| 4 | P2: root v0.6.0 released per CONTRIBUTING checklist | CHANGELOG cut `4c217e6`; FEATURES refreshed (6 fuzz targets, BOM matrix, sticky-ID, go-directive fix); worktree build+race green; annotated tag on CI-green commit; `go mod download @v0.6.0` from proxy OK; pkg.go.dev serves v0.6.0; `gh release create` (Latest); scratch consumer runs `KeyedLines` wire output verbatim |
| 5 | P3: ssetest v0.3.0 released (paired) | `d556e42`: require bumped to go-sse v0.6.0, `vendorHashSsetest` recomputed (fakeHash→build→real), stale flake comment fixed; CI green; worktree race green; tag pushed; proxy download OK; scratch consumer `Collect`+`RequireDataJSON` PASS; GH pre-release per ssetest precedent |
| 6 | P4: datastartest pairing verified | `datastartest/v0.3.0` (tagged `60cf5b1` by concurrent session): scratch consumer resolves root+datastartest+go-sse from proxy, compiles, serves `datastar-patch-elements` wire format E2E; local replaces in tagged go.mod verified inert for consumers (dependency replaces ignored) — hygiene note recorded in TODO_LIST |
| 7 | P5: go-datastar golden batch behind full gate | `a0c0aea` (landed by concurrent session) re-verified: workspace `-race` ×5 packages ok, isolation ×3 modules ok, vet+lint clean, erraudit ×3 zero violations, `go work sync` idempotent, no absolute replaces |
| 8 | P6: actionlint gate + devShell | `f8276ff`: `.github/workflows/actionlint.yml` on every push/PR (repo action pins), `pkgs.actionlint`+`pkgs.shellcheck` in devShell; first CI run green (33270124044) |
| 9 | P9: shellcheck proven, not assumed | planted SC2086 in scratch workflow → actionlint+shellcheck fails (exit 1); job step `command -v shellcheck` makes presence non-optional; all three real workflows pass clean with shellcheck active |
| 10 | P7: flake-update dry-run | `gh workflow run` → run 33270142954 green: update ok, drift=false, gate/close/PR/fail steps correctly skipped, zero open PRs left |
| 11 | P8: auto-close superseded update PRs | step added to `flake-update.yml` (close + `--delete-branch` + supersede comment before opening the new PR); `bash -n` + actionlint validated; fires on the first drift-bearing run |
| 12 | P10: conventions v1.1 | `docs/status/AGENTS.md`: TL;DR allowance, cross-repo cover-scope rule, specializes-global-format note, template link |
| 13 | P11: report template | `docs/status/_template.md` (title, preamble, cover line, TL;DR, a–g skeleton); linked from conventions + root AGENTS.md |
| 14 | D1 (P12): changelog policy ruling applied | top-level `## Changelog policy` in CHANGELOG codifies the precedent (CI gates/jobs/thresholds/schedules are changelog-worthy; deep chore is git-history-only); cron line stays in [0.6.0] Added |
| 15 | D2 (P13): `38e79aa` accepted | plan default applied — no history rewrite; AGENTS.md Gotcha remains the documentation; TODO_LIST carries a ⚪ WONT row with the reopen condition (explicit user order) |
| 16 | D3 (P14): browser-E2E stays blocked | plan default applied; TODO_LIST Blocked row updated with the default-applied note |
| 17 | Living docs synced | TODO_LIST rebuilt (only WONT/BLOCKED rows remain), CHANGELOG [Unreleased] carries the five new additions, AGENTS.md gains the lint-pin rule, template pointer, and the coverage-gate GOCACHE gotcha |

## d) TOTALLY FUCKED UP

1. The CHANGELOG cut used a multiedit whose second edit failed while the fourth applied against the half-edited state, silently merging two bullets and duplicating another (`4c217e6` predecessor state). Caught by re-reading the section immediately and repaired before commit — but the lesson repeats: for long-table restructures, `write` the whole file.
2. Two scratch-consumer failures were my own API mistakes (wrong `ssetest.Collect` signature, wrong `RequireElements` default mode, missing `testing` import), not release defects. I initially let a piped `tail` mask a real test exit code (`echo` ran after FAIL). Always check `$?` directly, never through a pipe.
3. Wasted a build cycle fetching the vendor hash with wrong flake output names (`.#ssetest`, `.#go-sse-ssetest`) before reading flake.nix and using `.#checks.x86_64-linux.build-ssetest`.
4. `gh release view --json isLatest` — guessed a nonexistent field; the CLI helpfully printed the valid list. Read the schema before querying.

## e) WHAT WE SHOULD IMPROVE

| IMP | Improvement | Priority |
| --- | --- | --- |
| IMP1 | Single-source the golangci-lint version between ci.yml and flake.nix (e.g. a comment cross-reference check in verify.sh, or bump both in one checklist step) so the 2.12/2.13 skew class cannot recur | high |
| IMP2 | Make `coverage-gate` hermetic against caller `GOCACHE` (export a sane default inside the app) — the silent exit-1 cost a debugging round trip | med |
| IMP3 | Add a scratch-consumer smoke script (`scripts/release-verify.sh <tag>`) encoding the CONTRIBUTING consumer check, so every future release runs the identical probe | med |
| IMP4 | go-datastar: drop local replaces before tagging (checklist discipline) — the tagged v0.3.0 go.mod carries inert-but-sloppy `replace => ..` lines | med |

## f) Up to 50 things we should get done next

1. IMP1: single-source the golangci-lint version (ci.yml ↔ flake.nix).
2. IMP2: `coverage-gate` own-GOCACHE hardening.
3. IMP3: `scripts/release-verify.sh` consumer probe.
4. Watch the first scheduled flake-update run (Monday 04:00 UTC) — it exercises the drift=true path (gate in-workflow, auto-close, PR) for real.
5. go-datastar next release: clean replaces from `datastartest/go.mod` before tagging (IMP4).
6. After Go 1.27: revisit the `stdversion` gopls friction (expected to disappear) and the `encoding/json/v2` experiment flag.
7. ssetest: consider requiring go-sse v0.6.0's features in a helper (e.g. a `RequireKeyedData` assertion built on `KeyedLines` round-trip) if consumers ask.

## g) Questions I CANNOT figure out myself

None this session. The one standing user decision (browser-E2E Option B vs C, D3) remains routed via TODO_LIST's Blocked row and the brainstorming doc — it is a scope decision, not a task.
