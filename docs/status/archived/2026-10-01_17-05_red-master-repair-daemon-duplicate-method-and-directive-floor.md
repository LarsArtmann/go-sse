# Status Report — 2026-10-01 17:05 — Red-master repair: daemon duplicate method + directive floor (3rd lowering)

Session opened with the mandated CI health probe instead of a feature task. Found master RED at HEAD `292c56f` (CI run `36880874852`: Lint, Vet, Nix flake check failing) plus a user-run `buildflow --fix --build-mode=full` that exited 69 with 6 failed steps. Diagnosed and fixed two daemon-introduced breakages, re-ran the full gate locally (`scripts/verify.sh`: ALL CHECKS PASSED, including hermetic `nix flake check`), let the daemon commit+push, and watched master CI return to green (run `36883370012`, 11/11 jobs). Docs updated (AGENTS.md, TODO_LIST.md). The buildflow gate itself was NOT re-run — see b). → done 2026-10-03 — re-run EXIT=0.

- cover: library 99.3% (=), ssetest 100.0% (=), sseparse 99.2% (=)

## TL;DR

- Master was red from daemon commit `292c56f`: a duplicate `eventBrand.Name` method (non-compiling) and a 3rd same-day re-lowering of the go directives to `1.27`.
- Both fixed in `4e50d31` + `e83fb52`; full local gate and master CI green again.
- The Nix failure looked like vendorHash drift but was the compile error; vendorHash `1G5uX…` was already correct.
- The session-opening `buildflow --fix` failure is only half-addressed: its lychee/shfmt/statix/meta findings remain open. → done 2026-10-03 — cleared in the flip-war closeout session (lychee, statix, flake-meta, pin comments).

## a) FULLY DONE

| # | Work                                                                                                                                                                                                      | Evidence                                                                                                                                                                                                                                                     |
| - | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1 | Session-start CI health probe; identified red master at HEAD with 3 failing jobs                                                                                                                          | `gh run list --workflow CI`; run `36880874852` (Lint X, Nix X, Vet X)                                                                                                                                                                                        |
| 2 | Removed daemon-injected duplicate `eventBrand.Name()` (returned `"event"`) at event.go:17, kept the documented `eventBrandName` method                                                                    | commit `e83fb52`; `go vet ./...` + `go build ./...` clean after; the duplicate was the root cause of all three CI failures incl. the Nix job (its `go: downloading error-family v0.11.0` tail line was log noise — vendorHash was already correct)           |
| 3 | Re-raised go directives `1.27` → `1.27.1` in root + `sseparse/go.mod` (3rd 2026-10-01 lowering, 5th overall; normalize-step trap)                                                                         | commit `4e50d31`; `go mod tidy` no-op in all three modules; verify.sh directive-equality gate passes                                                                                                                                                         |
| 4 | Full local gate green                                                                                                                                                                                     | `scripts/verify.sh` (full, not `--fast`): treefmt clean, tidy clean + directives aligned (`go 1.27.1` ×3), vet clean, golangci-lint clean (pin matches local v2.14.0), race tests ok ×6 packages, bench smoke pass, `nix flake check` → "all checks passed!" |
| 5 | Master CI green                                                                                                                                                                                           | run `36883370012` for `e83fb52`: all 11 jobs ✓ (Nix flake check 1m32s, Fuzz 6m53s)                                                                                                                                                                           |
| 6 | AGENTS.md updated: lowering count 2→3 on 2026-10-01 (`292c56f` cited); new daemon-failure-mode bullet — the daemon can commit NON-COMPILING code; run `go vet ./...` after any sweep touching `.go` files | commit `e83fb52`                                                                                                                                                                                                                                             |
| 7 | TODO_LIST.md: added ubuntu-latest → Ubuntu 26 runner-migration item (label switches 2026-10-19; 10 of 11 jobs exposed)                                                                                    | commit `af62866`                                                                                                                                                                                                                                             |
| 8 | Coverage measured this session (not quoted)                                                                                                                                                               | `nix run .#coverage-gate` → 99.3 / 100.0 / 99.2, all above thresholds, all `(=)` vs previous report line                                                                                                                                                     |

## b) PARTIALLY DONE

**Buildflow gate re-greening.** The session-opening `buildflow --fix --build-mode=full` run failed (exit 69): `test-race`, `golangci-lint`, `govalid-generate`, `go-mod-update` all failed _downstream of the duplicate method_ — those are now fixed by `e83fb52`/`4e50d31`. What remains, explicitly:

- buildflow was **not re-run** after the fixes, so "everything but lychee now passes" is inference, not measurement.
- **lychee will still fail it**: 8 broken links remain (see f/2).
- `go-mod-update` reverted its dep bumps because the tree didn't compile; a re-run on the fixed tree may produce real dependency bumps (plus the vendorHash dance if go.sum moves).

  → done 2026-10-03 — buildflow re-run EXIT=0 and all 8 lychee links fixed the same session; the go-mod-update re-run remains in the TODO_LIST buildflow-gate row.

**Daemon handoff.** Fixes were committed and pushed only via the daemon cycle (~5 min to commit, ~10 min to push). Outcome green, but master sat red ~75 minutes total (15:00 → 15:19 push → ~15:27 green) when a verified fix existed from ~17:45 local. Policy question in g/2. → routed — see the §g2 annotation (user policy; per-incident so far).

## c) NOT STARTED

Noticed during the session's probe/buildflow output, never began — all parked in f):

1. ~~The 8 lychee broken doc links (mostly archived status reports referencing moved files; `docs/status/archived/README.md` links to `docs/CHANGELOG.md` + `docs/TODO_LIST.md` which live at repo root).~~ done — 2026-10-03 — all 8 lychee broken links repaired (offline lychee: errors 0)
2. buildflow residual findings: statix ×13 (e.g. repeated `checks.format` key, flake.nix:194), flake-meta-checker ×9 (missing `homepage`/`mainProgram` ×3 packages), go-auto-upgrade warnings (sseparse `encoding/json` v1 boundaries ×3; `lo.Filter` suggestions ×2), oxlint's 179 warnings on the vendored `static/*.js` bundles, `lychee`/`vulnix` missing from devShell (nix-run fallback), `interrogate` not installed. **→ routed — statix repeated-keys + flake-meta + action-pin comments fixed 2026-10-03; the remainder (oxlint exclusion, devShell tools, vulnix policy) lives in the TODO_LIST buildflow-gate row**
3. CI run `36883370012` duration anomaly: reported 35m59s wall while no job exceeded ~7m (queue/concurrency artifact?) — uninvestigated; conclusion=success unaffected. **→ routed — TODO_LIST row (run 36883370012 wall-clock anomaly)**

## d) TOTALLY FUCKED UP

1. **First CI probe nearly missed the actual state.** `gh run list --branch master --limit 5` (all workflows) showed the 2026-09-29 failure as newest and hid every Oct-1 run including the in-progress run for HEAD. I initially framed the incident as "red since Sept 29" until the `--workflow CI` filtered call showed the truth. Always filter by workflow and compare head SHA.
2. **Misdiagnosed the Nix failure.** I theorized vendorHash drift and nixpkgs buildGoModule behavior changes (chased the `flake.lock` diff in `292c56f`) before reading the full job log. The grep excerpt (`go: downloading error-family v0.11.0`) was noise; the full log's compile error named the duplicate method directly. Full logs first, theories second.
3. **Sloppy final verification.** `git status -sb | head -1` truncated the dirty-file list, so the wrap-up claimed completion without actually confirming the TODO_LIST edit's state (daemon swept it later as `af62866` — right outcome, unverified check).
4. **Idled through the daemon wait.** ~12 minutes of poll loops while coverage (required by this report) could have been measured then.
5. **Edit-tool slip during the TODO_LIST harvest.** Swapped old/new content on the first edit, truncating the ubuntu-26 row mid-cell (`…2026-10-19) | GitHub migrates` → `…2026-10-19)`); caught via `git diff` and restored byte-identical before re-doing the insertion correctly. Net damage zero, process sloppiness real.

Neither breakage was authored this session — both rode in on daemon commit `292c56f` — but the session's whole job was to catch that class, and probe #1 almost reported the wrong incident.

## e) WHAT WE SHOULD IMPROVE

| IMP  | Improvement (process, not product)                                                                                                                                          | Priority |
| ---- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| IMP1 | Session-start probe: filter `--workflow CI` AND diff origin head SHA vs local HEAD before concluding anything                                                               | high     |
| IMP2 | Read the FULL failing CI job log before theorizing; grep excerpts actively mislead (this session: vendorHash ghost)                                                         | high     |
| IMP3 | Habit (now also AGENTS.md): `go vet ./...` after every daemon sweep that touches `.go` files                                                                                | high     |
| IMP4 | buildflow-side durable fix for the normalize step (derive target from `max(dep directives)`) — 5th lowering observed; detector works, generator still broken                | high     |
| IMP5 | Exclude vendored `example/*/static/*.js` bundles from oxlint (179 warnings on minified third-party code is config noise)                                                    | med      |
| IMP6 | Add `lychee` + `vulnix` to `devShells.default` so buildflow stops falling back to `nix run nixpkgs#…` without project deps                                                  | med      |
| IMP7 | Use daemon-wait windows for required measurements (coverage) instead of poll-sleeping                                                                                       | med      |
| IMP8 | Final checks: plain `git status --short`, never `head -1`-truncated status output                                                                                           | low      |
| IMP9 | Wrap the health probe as `scripts/ci-health.sh` (sha compare + workflow filter + annotation scan — the ubuntu-latest deprecation annotation sat unnoticed since 2026-09-29) | low      |

## f) Up to 50 things we should get done next

Grounded in this session's observations and the TODO_LIST read during it. Not re-researched.

**P1 — this week**

1. ~~Re-run `buildflow --fix --build-mode=full` on the fixed tree; confirm only lychee (and cosmetic steps) remain red.~~ done — 2026-10-03 — buildflow --fix --build-mode=full EXIT=0
2. ~~Fix the 8 lychee broken links: `docs/status/archived/README.md` → root `CHANGELOG.md`/`TODO_LIST.md`; archived reports `2026-08-29_18-25…`, `2026-08-16_11-58…`, `2026-08-07_00-44…`, `docs/planning/archived/2026-08-29_20-10…`, `docs/status/2026-09-29_05-49…` → moved-file targets; the empty-URL error at archived `2026-07-27_10-26…` line 123.~~ done — 2026-10-03 — errors 0 (six re-pointed + one quoted-empty-URL)
3. Ubuntu-26 migration decision + action before 2026-10-19 (TODO_LIST row, `af62866`). **→ routed — TODO_LIST row (decision + action before 2026-10-19)**
4. Record the deliberate decision on sseparse `encoding/json` v1 boundaries (gen_corpus.go, gen_corpus_roundtrip_test.go, wpt_format_corpus_test.go) — WONT with rationale, or migrate; the go-auto-upgrade warning will otherwise recur weekly. **→ routed — TODO_LIST buildflow-gate row (WONT-or-migrate decision)**
5. Decide the `lo.Filter` suggestions (`example/datastar/main_test.go:584`, `ssetest/e2e_test.go:21`) — likely WONT (samber/lo is not a dependency; adding it for one test filter is anti-lean). **→ routed — TODO_LIST buildflow-gate row (likely WONT — samber/lo not a dep)**
6. Re-run `go-mod-update` on the green tree (it self-reverted during the broken state); vendorHash dance if go.sum moves. **→ routed — TODO_LIST buildflow-gate row (re-run on the green tree)**
7. Investigate run `36883370012`'s 35m59s wall-clock vs ≤7m jobs (queue time? concurrency group?). **→ routed — TODO_LIST row (wall-clock anomaly)**

**P2 — near term**

8. ~~statix cleanups in `flake.nix` (13 findings; repeated `checks.format` key at :194).~~ done — 2026-10-03 — repeated checks keys grouped; statix check clean
9. ~~flake-meta-checker: add `homepage` + `mainProgram` to the 3 package meta blocks.~~ done — 2026-10-03 — homepage/mainProgram/platforms on all three check packages
10. Exclude `example/*/static/` from oxlint config. **→ routed — TODO_LIST buildflow-gate row (oxlint exclusion)**
11. Add `lychee`, `vulnix` (and `interrogate` if actually wanted) to `devShells.default`. **→ routed — TODO_LIST buildflow-gate row (lychee+vulnix in devShells.default)**
12. buildflow repo: implement IMP4 (normalize target from max(dep directives)) — kills the recurring lowering class at the source. **→ routed — TODO_LIST Blocked row (buildflow normalize target)**
13. vulnix advisory policy: build-time-only deps (binutils, ShellCheck, etc.) — pin an ignore-list or accept the noise consciously. **→ routed — TODO_LIST buildflow-gate row (vulnix advisory policy)**

**P3 — tracked rows re-confirmed open in TODO_LIST during this session**

14. go-datastar hygiene batch (single sweep, own repo — TODO_LIST row). **→ routed — TODO_LIST cross-repo row (go-datastar hygiene batch)**
15. pdd → go-sse v0.6.1 pin bump (repo not cloned locally; TODO_LIST row). **→ routed — TODO_LIST consumer-pin row (pdd → go-sse v0.6.2)**
16. Bump `minCorpusVectors` (25) at next real WPT ingestion (trigger-gated row). **→ routed — TODO_LIST trigger row (minCorpusVectors)**
17. Re-verify AGENTS go-directive claims at next root tag (trigger-gated row). **→ routed — TODO_LIST trigger row (directive claims at the next root tag)**

## g) Questions I CANNOT figure out myself

1. **Ubuntu-26:** pin `ubuntu-24.04` in ci.yml now (deterministic, one boring PR) or ride `ubuntu-latest` through the 2026-10-19 switch and verify after (fresh, riskier)? This is a stability-vs-freshness policy call; the repo's history (vendorHash/lint-pin/format drift) argues for pinning, but that forecloses security updates in the image. **→ routed — TODO_LIST ubuntu-26 row (pin-or-ride decision, owner call)**
2. **Red-master push policy:** when master is red and the fix is locally gate-verified, do you want me to commit+push immediately — explicitly overriding the never-commit/never-push defaults — or always defer to the daemon cycle (this session: ~75 min of avoidable red)? **→ routed — user policy; per-incident so far (pushes flowed daemon/user-side on 07-03 and 10-11)**
3. ~~**shfmt canonical style:** buildflow's shfmt repair proposes 4-space indentation for `scripts/*.sh`; the committed canonical style (treefmt + CI format gate) differs. Which is authoritative — fix buildflow's shfmt config to match the repo, or reformat the repo to buildflow's style (one noisy commit, then stable)?~~ done — resolved 2026-10-03 — buildflow shfmt step skipped in .buildflow.yml; treefmt 2-space stays canonical
