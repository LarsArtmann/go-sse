# Status Report — 2026-11-22 23:48 — erraudit strict-audit disposition & vendored-bundle repair

**Session scope:** disposition the `erraudit lint ./... --max` findings (1 ERROR in vendored JS,
11 Go generic-return advisories), record the strict-audit posture, and evaluate a mid-session
proposal to move `example/datastar/` into go-datastar.

**Verified this report:** `scripts/verify.sh --fast` → ALL CHECKS PASSED (treefmt, vet,
golangci-lint ×3 modules at pin v2.15.0, race tests ×6 packages, tidy-clean, bench smoke);
`nix run .#coverage-gate` → OK; `erraudit lint ./...` exit 0; `erraudit lint ./... --max
--disable generic_return` exit 0; `erraudit fix ./...` dry-run → "No fixes found" ×3 modules.
**NOT verified:** `nix flake check` (skipped by `--fast`; no go.mod/go.sum/flake.nix inputs
changed, so vendorHash risk is nil); CI health (`gh run list` produced no output — probe
not completed); remote state of the fix commit.

- cover: library 100.0% (=), ssetest 100.0% (=), sseparse 100.0% (=) — measured this session
  via `nix run .#coverage-gate`, all three at the gate baseline
- The fix commit `dadb670` (AGENTS.md bullet + `example/datastar/static/datastar.js`) is
  **local only** (master ahead 1, unpushed) — verdict hashes citing it resolve locally, not
  in a fresh clone, until pushed.

## a) FULLY DONE

| #  | Work                                                                                                                                              | Evidence                                                                                                                            |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| a1 | Vendored `datastar.js` empty-catch ERROR fixed with an in-body intent comment (the checker's documented-ignore remedy, honored even under `--no-suppress`) | `example/datastar/static/datastar.js:1354`; `erraudit lint ./...` → exit 0; erraudit source `internal/jserr/detectors.go:119` confirms comment-only bodies are intentional ignores |
| a2 | 11 `--enforce-generic-return` advisories triaged and ACCEPTED (deliberate non-fixes: public error APIs return idiomatic `error`; callers classify via go-error-family codes with `errors.Is`; concrete types = API break + typed-nil landmine) | `erraudit lint ./... --max --disable generic_return` → exit 0; `erraudit fix ./...` dry-run → "No fixes found" in root, ssetest, sseparse |
| a3 | Strict-audit posture recorded in AGENTS.md Gotchas (strict gate = `--max --disable generic_return`; no retyping; no `//nolint` — `--max` bypasses directives; re-add the catch comment on bundle upgrades) | AGENTS.md erraudit bullet (commit `dadb670`, local)                                                                                  |
| a4 | Full pre-push gate re-run after the changes                                                                                                       | `scripts/verify.sh --fast` → ALL CHECKS PASSED (0 lint issues ×3, all tests ok, benchmarks smoke-tested)                              |
| a5 | Coverage measured fresh for the report                                                                                                            | `nix run .#coverage-gate` → library 100.0%, ssetest 100.0%, sseparse 100.0%, all at baseline                                          |

## b) PARTIALLY DONE

- **erraudit posture enforcement:** disposition recorded in AGENTS.md prose (done), but NOT
  mechanically enforced — no verify.sh lane, no CI job, no script. The strict gate currently
  exists only in documentation; a future session that forgets the bullet re-runs bare
  `--max` and rediscovers the 11 advisories from zero.
- **Report → harvest loop:** this report written (done); harvest of section f) into
  `TODO_LIST.md`/`ROADMAP.md` NOT yet done — per docs/status conventions it is the immediate
  next step, held for user instruction.
- **Vendored-bundle divergence:** the catch comment is a local edit to vendored upstream
  code (done); it is NOT upstreamed to DataStar and will be silently lost on the next
  bundle upgrade unless re-added (AGENTS.md documents the re-add requirement).

## c) NOT STARTED

- **CI health probe** — AGENTS.md mandates probing `gh run list --branch master` at session
  start; the command produced no output this session and was not retried.
- **Push** — `dadb670` unpushed (deliberately: pushing is never done without an explicit
  request).
- **`nix flake check`** — skipped by `--fast`; not separately run.
- **CHANGELOG [Unreleased]** — no entry yet for the vendored-bundle lint fix.
- **datastar-example move** — evaluation done this session (see section g, question 1); no
  move/rewrite work started.

## d) TOTALLY FUCKED UP

1. **Pipeline exit-code sloppiness.** `erraudit lint ./... | tail -15; echo "EXIT=$?"` printed
   `EXIT=0` while the run was RED — `$?` captured `tail`'s exit status, not erraudit's. The
   conclusion was unaffected (the finding text itself was in view), but the printed evidence
   was wrong, and the same pattern could mask a real failure in a different session. Correct:
   capture erraudit's exit before the pipe, or run unpiped, or use `PIPESTATUS`.
2. **Skipped the mandated session-start probe.** AGENTS.md says probe CI health (`gh run list
   --branch master --limit 5`) at session start — master has repeatedly sat red while local
   trees looked clean. Not done, and the retry produced empty output that went unquestioned.

## e) WHAT WE SHOULD IMPROVE

| IMP# | Priority | Improvement                                                                                                                                                             |
| ---- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| IMP1 | P0       | Make the erraudit posture mechanical: a `scripts/` lane (or verify.sh step) running `erraudit lint ./... --max --disable generic_return` so the strict gate is one command, not remembered prose |
| IMP2 | P1       | Vendored-bundle edit loss-prevention: on any DataStar bundle upgrade, re-add the catch comment (AGENTS.md documents it) — or upstream the comment so upgrades carry it natively |
| IMP3 | P1       | Evidence discipline: never capture exit codes from behind a pipe; assert the tool's exit directly                                                                        |
| IMP4 | P2       | erraudit binary has moved (`d19cca1e` → `ee8a415`) since the go-error-modernization skill's last flag verification; a flag-table refresh belongs in the crush-config repo |
| IMP5 | P2       | Session-start ritual: make the CI probe + ahead/behind check part of the opening tool batch, per AGENTS.md                                                              |

## f) Up to 50 things we should get done next

Honest bounded list (18 items, grouped by priority) — session-sourced; harvest must verify
each against code and the existing TODO_LIST before routing.

**P0 — direct follow-ups of this session**

1. Decide question g1 (datastar example: keep vs move) — blocks items 8 and 9.
2. Decide question g2 (erraudit mechanical gate or manual-only) — blocks item 13.
3. Harvest this report into `TODO_LIST.md` (bounded items) and `ROADMAP.md` (vague ones), verifying against code first — docs-gates freshness expects it.
4. Re-run the CI health probe with output captured (`gh run list --branch master --limit 5`) — the mandated session-start check never produced evidence.
5. Push (or deliberately hold) `dadb670` — until pushed, the AGENTS.md posture bullet cites a commit no fresh clone can resolve.

**P1 — small, bounded**

6. Add the CHANGELOG `[Unreleased]` row for the vendored-bundle lint fix.
7. Check DataStar bundle currency (vendored v1.0.2 vs upstream releases); on any upgrade, re-add the in-body catch comment (AGENTS.md).
8. If g1 lands "keep": reframe `example/README.md`/datastar example docs as "go-sse feature demo in DataStar's wire dialect" to resolve the payload-flavor scope question.
9. If go-datastar should demo an app: add a NEW example there using the Patch API (it already has `*_example_test.go` files) — do NOT transplant this one (it imports only go-sse).
10. Land the stdlib `net/http` GO-2026-6859 fix via the next nixpkgs bump (weekly flake-update PR) and verify the devShell pins Go ≥ 1.27.4.
11. vulnix's 17 nix-store build-input advisories: same nixpkgs-bump path.
12. Run `nix flake check` once (skipped this session via `--fast`).

**P2 — maintenance / external repos**

13. Implement IMP1: an erraudit strict-gate script/lane.
14. Upstream the one-line catch comment to DataStar (see g3).
15. Re-verify the standing TODO_LIST Blocked policy row (Actions-created PRs run CI only after human approval) is still current.
16. Update the go-error-modernization skill (crush-config repo, committed there) with ee8a415-era facts: the empty_catch in-body-comment remedy and `--modules all` merged multi-module behavior.
17. Sanity-run erraudit from inside `ssetest/` once to confirm the per-module posture matches the merged `--modules all` result (ssetest had zero findings in the merged run).
18. If IMP1 lands, consider `erraudit --format json` for machine-readable gate output.

## g) Questions I CANNOT figure out myself

1. **The datastar example — keep in go-sse, or move to go-datastar?** Measured fact: `example/datastar/` imports ONLY `github.com/larsartmann/go-sse` — it never touches the go-datastar library; it writes DataStar's wire dialect through go-sse's keyed-line helpers, and its footer says "Powered by go-sse". Moving it as-is would plant a demo in go-datastar's repo that never uses go-datastar (a ghost system), plus two-repo CI/docs churn. My recommendation: **keep it in go-sse** — it is the only example exercising every transport feature (fan-out, replay, filtering, heartbeat, keyed format) and is linked from README — and if go-datastar should demo an app, write a new Patch-API example THERE (item 9). If scope purity wins instead, the move should be paired with a Patch-API rewrite, which is a project, not a move. Your call.
2. **Should erraudit become a mechanical gate (verify.sh lane or CI job), or stay a manual audit?** A gate enforces the posture every session; manual-only costs zero CI time but relies on the AGENTS.md bullet being remembered. My lean: a local verify.sh lane (fast, no CI cost), not a CI job.
3. **Upstream the one-line catch comment to the DataStar project** (so future bundle upgrades carry it natively), or keep it as a local, AGENTS.md-documented divergence? An upstream PR for a linter-appeasement comment may not be worth upstream goodwill — I cannot judge that tradeoff for you.
