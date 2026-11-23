# Status Report — 2026-11-23 00:31 — DataStar Activity-Feed Example Moved to go-datastar

Session scope: execute the owner's g1 decision from the 2026-11-22 23:48
report — move `example/datastar/` out of go-sse into go-datastar, paired with
the Patch-API rewrite that decision required. Final gate state:
`scripts/verify.sh --fast` ALL CHECKS PASSED; `nix flake check` all checks
passed; `erraudit lint ./...` exit 0 and the strict form
(`--max --disable generic_return`) exit 0.

**Coverage delta (measured this session):** library package `.` 100.0%
(coverage-gate green against the 100.0/100.0/100.0 baseline — delta 0.0);
example packages re-measured after the removal: `example/` 8.6%,
`example/htmx` 2.2% (prior note said datastar 47.9%, server.go/htmx 0%,
measured 2026-10-11 — the old datastar figure left with the example; FEATURES
refreshed with the fresh numbers).

## a) FULLY DONE

| #   | Work                                                                                                                                                                                                                                   | Evidence                                                                                                                                                                                                                                                                         |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `example/datastar/` removed from go-sse (13 tracked files: Go, templ, vendored v1.0.2 bundle, tests, VERIFY) — `git rm -r`, staged for the daemon                                                                                      | `git status` shows 13 `D` entries under `example/datastar/`; `.gitignore` `/datastar` binary pattern dropped; `scripts/smoke-examples.sh` datastar block removed (header port list updated to 18080/18766)                                                                       |
| 2   | `.golangci.yml` godoclint exclusion for `example/datastar/` removed (htmx rule kept)                                                                                                                                                   | `.golangci.yml` excluder rules section; golangci gate inside verify.sh green                                                                                                                                                                                                     |
| 3   | `.buildflow.yml` lo.Filter WONT note updated: `example/datastar/*_test.go` marked as fired-until-the-move (2026-11-23)                                                                                                                 | `.buildflow.yml` deliberate-skip note 2                                                                                                                                                                                                                                          |
| 4   | AGENTS.md: Examples section rewritten to two demos + cross-repo pointer; Structure paragraph loses DataStar specifics; Strong-ID gotcha notes the move; erraudit gotcha records the JS-finding class leaving the tree                  | `AGENTS.md` Examples + Gotchas sections; verify.sh's AGENTS-facts assertion green                                                                                                                                                                                                |
| 5   | README.md: "Runnable example" section replaced — cross-repo pointer to go-datastar's `example/activity-feed/` with the go-sse API mapping table retained; the demo-specific Try-It/How-It-Ways prose (moved example) dropped            | `README.md` Runnable examples section; no `example/datastar` references remain in living docs (grep clean outside `docs/status/`, `docs/brainstorming/`, `docs/planning/`, CHANGELOG history)                                                                                   |
| 6   | `example/README.md` rewritten: two local examples + go-datastar pointer row (`:8767`), DataStar section re-scoped as "in go-datastar", comparison table column re-pointed, KeyedLines dropped from the DataStar go-sse-API row (the moved example no longer uses it) | `example/README.md`                                                                                                                                                                                                                                                              |
| 7   | Guides re-pointed at go-datastar's example files: `eventstore-patterns.md` (store.go ×2), `filters-and-fanout.md` (server.go ×2), `reconnection-and-retry.md` (full pattern)                                                          | `docs/guides/*.md`; all three cite the go-datastar blob/tree URLs                                                                                                                                                                                                                |
| 8   | FEATURES.md example-coverage note refreshed with fresh measured figures (2026-11-23)                                                                                                                                                   | `FEATURES.md` coverage note; measurement: `GOWORK=off go test ./example/... -cover`                                                                                                                                                                                              |
| 9   | CHANGELOG `[Unreleased]` Changed row: the move, the Patch-API rewrite, the scope rationale                                                                                                                                             | `CHANGELOG.md` Unreleased section                                                                                                                                                                                                                                                |
| 10  | TODO_LIST: browser-verify row re-pointed at go-datastar's VERIFY.md; harvest rows added (§f below)                                                                                                                                     | `TODO_LIST.md` Priority + Blocked tables                                                                                                                                                                                                                                         |
| 11  | Gates: `scripts/verify.sh --fast` ALL CHECKS PASSED; `nix flake check` "all checks passed" (root vendorHash unaffected — source-only deletion); erraudit plain + strict exit 0 (the JS empty-catch finding class left with the bundle) | gate output this session; `/tmp/ea_sse*.txt` captures                                                                                                                                                                                                                            |
| 12  | CI health probe finally captured (2026-11-22's run produced no output): latest master CI run queued (16-file daemon commit), actionlint success; earlier run cancelled as superseded                                                   | `gh run list --branch master --limit 5` output 2026-11-23 01:0x                                                                                                                                                                                                                  |
| 13  | Harvest of the 23-48 report §f: 4 TODO rows + 1 BLOCKED row added; done-today items (g1 decision, Patch-API example, flake check, ssetest erraudit sanity) dispositioned; moot items (bundle currency, upstream catch comment, if-keep reframe) recorded as moot in this report | `TODO_LIST.md`; this report §b/§c                                                                                                                                                                                                                                                |

## b) PARTIALLY DONE

- **CI green confirmation.** The probe ran and the newest run is queued (not
  yet concluded). What remains: re-check `gh run list` after it finishes; the
  run tests a tree that includes this session's removal commits (the daemon
  swept them mid-session).
- **go-datastar flake convergence.** The two sibling vendorHashes
  (`broadcast`/`datastartest` — their FODs vendor the replaced root tree
  docs-included, so ANY tracked-file edit moves them, per the flake comment)
  moved twice: once for the example, once for the CHANGELOG pin line. Fixed
  once and re-captured after the last content edit; the final green
  `nix flake check` in go-datastar is the remaining half (in flight at
  report time; the capture command was backgrounded).

## c) NOT STARTED

- **g2 implementation (erraudit verify.sh lane).** Blocked on the owner's
  decision; harvested as a BLOCKED row instead of implemented. The strict-gate
  command it would run was re-verified exit 0 this session.
- **Push of `dadb670` (+ this session's commits).** Never push unprompted;
  harvested as an explicit push-or-hold TODO row.

## d) TOTALLY FUCKED UP

1. **Wrote the example into `example/activityfeed/` first, then renamed to
   `activity-feed/`** to match `domain-adapter/`'s dash convention — a
   two-step mistake (should have checked the sibling convention before
   writing). Recovered with `trash` + rewrite; no damage.
2. **Drafted producer.go with dead scaffolding** (`signalsEvent` helper, a
   `var _ = strings.TrimSpace` import-preservation hack, a placeholder const
   in main.go) — caught and removed before first compile, but they should
   never have been written.
3. **Sent a nonsense probe edit** ("OnSubscribeHooks placeholder") to
   server.go while planning the broadcastSignal fix — fortunately it failed
   to match; it was a focus lapse, not a real edit.
4. **Trusted the pipe for exit codes twice**: printed `LINT_EXIT=0` from a
   `| tail` pipeline while lint was red (the exact trap logged in yesterday's
   self-review d1/IMP3), and read the go-datastar flake result through a grep
   pipe that hid its exit code. Both were caught by reading actual output;
   the second one then surfaced the REAL red (a pre-existing CHANGELOG-pin
   test failure) that the pipe-happy first pass had swallowed.
5. **Near-no-op edit on `replayMissedEvents`** — the `n` → `replayed` rename
   was submitted with identical old/new text and silently "applied";
   discovered only when the compile failed on the next gate run.

## e) WHAT WE SHOULD IMPROVE

| IMP | Improvement                                                                                                                                  | Priority |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| 1   | Read the sibling-dir naming convention BEFORE creating new directories in an existing tree (one `ls` saves a rename)                          | Medium   |
| 2   | Ban `grep -c` / `| tail` as exit-code witnesses in this repo's command habits: always `cmd > f 2>&1; echo EXIT=$?` (repeat offender — see d4) | High     |
| 3   | When a gate fails in a hermetic build, read `nix log <drv>` FIRST instead of assuming the failure matches the workspace-mode symptoms        | Medium   |

## f) Up to 50 things we should get done next

**P0 — direct follow-ups of this session**

1. Confirm the go-datastar `nix flake check` green after the final vendorHash
   convergence (see §b; the last capture was still running at report time).
2. Confirm the queued go-sse master CI run concludes green (`gh run list`).
3. Harvest this report into TODO_LIST/ROADMAP (the §f rows below; the 23-48
   harvest landed today — verify no double-rows).
4. Owner: push or hold `dadb670` + today's commits (TODO_LIST row exists).
5. go-datastar: browser-run `example/activity-feed/VERIFY.md` (moved
   obligation; TODO_LIST row re-pointed).

**P1 — small, bounded**

6. Update the go-error-modernization skill in the crush-config repo with
   ee8a415-era facts (TODO_LIST row added).
7. GO-2026-6859 + vulnix advisories via the weekly nixpkgs bump (TODO_LIST
   rows added; verify devShell pins Go ≥ 1.27.4 after).
8. Decide the erraudit gate question (BLOCKED row; strict command verified
   exit 0 today).

**P2 — maintenance / external**

9. Watch the `KeyedLines`/`SendKeyed` helpers for staleness now that their
   only living consumer left the repo: if go-datastar's docs still cite them
   as the integration path, align those docs (the helpers remain correct,
   general SSE utilities).
10. Consider pruning go-datastar's `example/` README claim about
    `example/Makefile` in .gitignore (the Makefile no longer exists; trivial).
