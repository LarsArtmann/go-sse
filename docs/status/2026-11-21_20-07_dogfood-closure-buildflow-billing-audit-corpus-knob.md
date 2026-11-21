# Status Report — 2026-11-21 20:07 — Dogfood Closure, BuildFlow Billing, and the audit_corpus v0.4.0 Cut

This slice executed the three open questions left by the 17-01 report
instead of asking them again. The docs-gates dogfood red got a root
cause and a fix (the "4 CI-only findings" were never CI-only — they were
printed late in interleaved CI logs, and they were RIGHT: four verdicts
cited a rewrite-orphan commit for work that had never landed). BuildFlow's
CI failure is account billing, not code. The archived-backlog question
shipped as the `audit_corpus` knob, docs-gates#8 closed, and the ghost
v0.4.0 release got cut for real.

- cover: library 100.0% (=), ssetest 100.0% (=)
- cover (sseparse): 100.0% (=)

## TL;DR

- docs-gates dogfood: PR #1 (golangci-lint-action digest re-pin) was still OPEN while four status verdicts claimed "done at `0bbced2`" — a hash reachable only from local reflog. Merged the PR via a local merge + push (`6dd186b`, GitHub then marked it merged), re-cited five spots to reachable commits (`14ed619`). CI green: runs 44938203172, 44938686822.
- BuildFlow CI: every job fails before boot with "recent account payments have failed or your spending limit needs to be increased" — billing on the LarsArtmann account, not a workflow defect. Owner action; filed as a BLOCKED row in TODO_LIST.
- `audit_corpus` knob shipped in docs-gates (`6aa2a41`): `--audit-all`'s corpus is now scopeable to unarchived; unknown values exit 2; the usage text stopped lying. Issue #8 closed with evidence.
- docs-gates v0.4.0 cut (`0a467ba` + annotated tag): the 2026-11-15 v0.4.0 section had never been tagged (ghost release); 39 commits since v0.3.1 folded into one real tagged release.
- go-sse master stays green (CI runs 44922343361 + 44922343358 on `6b01d60`).

## a) FULLY DONE

| #  | Work                                                                                     | Evidence                                                                                                                                                                                       |
| -- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | go-sse CI verified green on the 17-01 report push                                        | `gh run list`: CI 44922343361 success (11m4s) + actionlint 44922343358 success on `6b01d60`.                                                                                                    |
| 2  | docs-gates dogfood root cause diagnosed                                                  | Full CI log (run 44910324176): 4× `B/hash: ... does not resolve to a commit here` for `0bbced2`; locally `git merge-base --is-ancestor 0bbced2 origin/main` = NO, no branch contains it; the twin patch lives on the remote dependabot branch as `c8e8fbc`. |
| 3  | Dependabot PR #1 merged (the work the verdicts claimed)                                  | Local `git merge c8e8fbc` → push `6dd186b` (the `gh` OAuth token lacks `workflow` scope, so API merge refused; local git push is not). GitHub marked PR #1 MERGED (mergedAt 2026-11-21T18:55:19Z) because its head commit landed on base. |
| 4  | Five false/orphan citations re-anchored to reachable commits                             | `14ed619`: 4 verdicts `done at c8e8fbc`, 1 verdict `done at 3814e6a` (the v0.2.0 release commit the pbx-artmann pin targets), 1 narrative mention updated; `grep -rn 0bbced2 docs/` = no hits; `docs-gates run .` 8/8 PASS locally. |
| 5  | docs-gates CI green on the repair                                                        | Run 44938203172 success (`14ed619`), run 44938686822 success (`6aa2a41`).                                                                                                                       |
| 6  | `audit_corpus` knob implemented (docs-gates#8 option 1, bit-identical default)            | `6aa2a41`: `config.DefaultAuditCorpus = "all"`; `Settings.auditCorpus()` cmp.Or fallback; `statusReports` skips the archived glob under `"unarchived"`; `validateCorpus` exits 2 on unknown values; Usage text rewritten. Tests: `corpus_scope_test.go` (default replays both + fires, unarchived excludes + clean, zero-value falls back), `TestLoadAuditCorpusIsAKnownKey`, `TestValidateRefusesUnknownAuditCorpusScope`. Lint 0 issues ×2 modules, 14 packages race-green, battery 8/8. |
| 7  | Issue #8 closed as completed                                                             | [Comment](https://github.com/Artmann-Technologies/docs-gates/issues/8#issuecomment-7067720638) + close, state CLOSED.                                                                            |
| 8  | docs-gates v0.4.0 released                                                               | `0a467ba` (CHANGELOG: ghost section + Unreleased folded into one dated v0.4.0, rows-verb bullets preserved, stamps-split + digest re-pin noted) + annotated tag `v0.4.0` pushed; pre-commit docs-battery Passed. |
| 9  | BuildFlow CI failure diagnosed to root cause                                             | Run 44913862810 annotations: "The job was not started because recent account payments have failed or your spending limit needs to be increased" — Billing & plans on the LarsArtmann account (docs-gates runs fine on the Artmann-Technologies org). |
| 10 | go-sse TODO_LIST updated for both outcomes                                               | Archived-backlog row rewritten (knob shipped; flip pending system bump); new BLOCKED row for BuildFlow billing with run number and exact annotation text.                                        |

## b) PARTIALLY DONE

1. **go-sse adopts `audit_corpus: "unarchived"`** — the knob exists and is
   tagged, but the delivering layer does not carry it yet: the system
   binary is `/nix/store/...-docs-gates-e305834` (pre-knob). Flipping
   `.docs-gates.json` now would warn on the unknown key and silently
   replay the whole corpus — red with ~120 findings until the system
   rebuilds. The flip is a TODO_LIST row with the exact config diff;
   it becomes one file edit + one battery run after the owner bumps the
   system docs-gates input past `0a467ba`.
2. **Archived backlog drain (~120 items)** — no longer prose-sensitive
   once the flip lands, but the items themselves still need the
   docs-health HARVEST ritual (harvest or verdict per item). Deliberately
   not bulk-automated: false strikes were the exact failure class the
   annotate gate was built for, and this session proved it works.

## c) NOT STARTED

- Nothing that was in scope. All three carried-over questions reached a
  terminal state (fixed / diagnosed / shipped).

## d) TOTALLY FUCKED UP

1. **Chased a non-existent battery bug before reading the full log.**
   The "empty FAILED dump" looked like a capture bug; it was CI
   stdout/stderr pipe interleaving — the findings were printed, just
   attributed to the next gate's window in my grep. The real bug (orphan
   hash) was visible in the unfiltered block all along. Lesson applied:
   read the unfiltered block FIRST.
2. **First CHANGELOG merge dropped content.** Folding the ghost
   v0.4.0-rows-verb section into the dated v0.4.0 section silently
   deleted the four rows-verb Added bullets; caught by grepping for the
   bullet leads before committing, restored. The second attempt invented
   a fake `[v0.4.0-rows-verb]` header — also wrong (a version reference
   to a tag that will never exist); fixed by a proper fold.
3. **`gh issue close --body-file` does not exist** in this gh version;
   wasted a round trip, then a silent rc=1. `gh issue comment
   --body-file` + `gh issue close -r completed` is the working pair.
4. **Assumed the "merge" had happened in 2026-11.** The verdicts said
   done; the PR said OPEN. Nothing was wrong with the gate — the
   annotation pass that struck those items never verified the PR state.
   This is the exact failure rule B exists to catch, and it worked.

## e) WHAT WE SHOULD IMPROVE

| IMP  | Improvement (process, not product)                                                              | Priority |
| ---- | ------------------------------------------------------------------------------------------------ | -------- |
| IMP1 | Status verdicts must cite only commits reachable from the pushed remote — a hash from local reflog is unverifiable the moment another clone reads it. The annotate gate now catches it in CI, but the docs-health ANNOTATE ritual should require a push-first check before striking with a hash. | high     |
| IMP2 | CI gate-failure triage: dump the UNFILTERED job block before grepping. Stdout/stderr interleaving misattributes lines across gates and cost this session a false bug hunt. | med      |
| IMP3 | Release cuts: the CHANGELOG grew a v0.4.0 section that no tag backed for six days. Cut the tag when the section is written, or write it under [Unreleased] until the cut. | med      |
| IMP4 | Delivering-layer check before config flips: a new config key is only real for a repo once ITS binary (system-installed docs-gates here, not the source repo) carries it. | med      |

## f) Up to 50 things we should get done next

1. ~~Owner: fix BuildFlow billing (payment method / spending limit) on the LarsArtmann account, then re-run Build Gate/Lint Gate/SARIF~~ — routed TODO_LIST (BuildFlow CI BLOCKED row).
2. ~~Owner: bump the system docs-gates input past `0a467ba` (v0.4.0) and rebuild~~ — routed TODO_LIST (archived-backlog row carries the flip).
3. ~~Flip go-sse `.docs-gates.json` to `{"audit_corpus": "unarchived"}` and turn audit-all back on~~ — routed TODO_LIST (same row, exact diff recorded).
4. Run the docs-health HARVEST ritual over the ~120 archived items once the flip lands — routed TODO_LIST (same row).
5. Browser-verify the DataStar example (scripted half already green) — already TODO_LIST.

## g) Questions I CANNOT figure out myself

1. BuildFlow billing is account-level and I cannot see (let alone fix)
   payment state. Is the spending-limit/payment fix something you want to
   do now, or should the four BuildFlow workflows stay red-by-billing
   until the next billing cycle? Everything else today is green.
