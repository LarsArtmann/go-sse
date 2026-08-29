# Status Report — 2026-08-29 19:08 — TODO-List Follow-Up: Cron Workflow, Report Conventions, Writer Goldens

Session scope: executed the four remaining 🔴 items of the 2026-08-29 TODO_LIST follow-up
(the weekly flake-update cron, both docs items, and the go-datastar writer goldens), plus
re-verified the stale cross-repo premises. Work spans two repos: go-sse (workflow + docs)
and go-datastar (golden test). Everything verified green in the working trees.

Final gate state at report time: `scripts/verify.sh --fast` → `ALL CHECKS PASSED`
(vet + lint 0 issues both modules + race tests root/example/ssetest);
`nix run .#coverage-gate` → `OK`; `actionlint` (borrowed from the go-datastar devShell)
→ clean on the new workflow; go-datastar root module → vet clean, `golangci-lint run .`
→ 0 issues (incl. gofumpt/golines/gci formatters), `go test . -race -count=1` → ok.

- cover: library 99.3% (=), ssetest 97.2% (=) — measured via `nix run .#coverage-gate` this session

---

## a) FULLY DONE

Verified complete, evidence-backed. Citations are working-tree paths and gate results;
the auto-git daemon commits async, so hashes may land after this report.

| # | Work | Evidence |
|---|------|----------|
| 1 | Weekly `nix flake update` + full-gate cron workflow: Mondays 04:00 UTC, bumps `flake.lock`, runs `nix flake check` in-workflow on the bumped inputs (continue-on-error + outcome captured), opens a review PR even on a red gate so the failing diff is actionable, and a final step reddens the run when the gate failed | `.github/workflows/flake-update.yml`; YAML parsed (triggers/permissions/step conditions verified programmatically), embedded shell `bash -n` clean, `actionlint` clean |
| 2 | The in-workflow-gate design decision is documented in the workflow itself: GITHUB_TOKEN-opened PRs never trigger `pull_request` workflows (GitHub recursion guard), so the gate cannot live on the PR | `.github/workflows/flake-update.yml` comments; mirrors the PR body the workflow generates |
| 3 | Auto-git commit-message review Gotcha added to AGENTS.md, with the `38e79aa` counter-example and the after-the-fact review rule | `AGENTS.md` Gotchas (after the `nolint` bullet) |
| 4 | `docs/status/AGENTS.md` created: file-naming rule, the a–g section skeleton (derived from the three non-archived reports' actual shapes), annotation/harvest rules, `archived/` policy, and the **mandatory coverage-delta line** with rationale (the 2026-07-27 100% → 99.5% miss) | `docs/status/AGENTS.md`; root `AGENTS.md` Gotcha points to it |
| 5 | Writer goldens for DataStar patches in go-datastar core: `TestPatchWireGoldens`, 12 cases pinning exact `sse.WriteEvent` bytes for every patch family (elements default/selector+mode/full-options/remove-sugar, signals default/onlyIfMissing+id+retry, script default/attributes+autoRemove-off, redirect, console-log, custom-event default/selector+inverted-flags) | `go-datastar/wire_golden_test.go`; generated from a scratch dumper, each output reviewed against the DataStar protocol before pinning; all 12 subtests PASS |
| 6 | The goldens pin the subtle wire facts, reviewed one by one: data-line ordering (selector→mode→namespace→useViewTransition→viewTransitionSelector→elements), default elision (outer/html/default-retry never emitted), `id:`→`retry:` field order after data lines, one `data:` line per source line for multi-line payloads, and the **trailing-space `data: elements ` line** a blank JS line produces | `go-datastar/wire_golden_test.go` case comments; `go-sse/event.go:148-192` (WriteEvent order) |
| 7 | Stale TODO premise corrected: the datastartest parity batch was NOT sitting uncommitted — the daemon had committed it (`d032dc5`, `7fa8ed4`). The remaining release-tag step is now correctly 🔵 BLOCKED on the ssetest release pairing rule, not 🔴 | `git -C go-datastar log -- datastartest/`; TODO_LIST Cross-repo section |
| 8 | Living docs synced: TODO_LIST rebuilt (3 TODO rows + 1 stale row removed, 1 genuinely-new TODO added, 1 row converted to BLOCKED), CHANGELOG `[Unreleased]` +3 lines (go-sse) and +1 (go-datastar) | `TODO_LIST.md`, `CHANGELOG.md`, `go-datastar/CHANGELOG.md` |
| 9 | New TODO discovered and routed: go-sse has no actionlint anywhere (go-datastar runs it always-on since `5887043`); this session's workflow was linted with a borrowed devShell instance | `TODO_LIST.md` CI & tooling row |

## b) PARTIALLY DONE

None — every item attempted this session either finished or was re-classified (see a7).

## c) NOT STARTED

The three BLOCKED items (browser-E2E scope decision, `38e79aa` amend-vs-accept,
datastartest tag release) all require user decisions or a prior ssetest release;
none can proceed from a session. Tracked in TODO_LIST `Blocked`/`Cross-repo`.

## d) TOTALLY FUCKED UP

1. **I wrote a comment about the trailing space and then omitted the trailing space.**
   The golden for the custom-event's blank JS lines needed `data: elements ` (with
   trailing space); my first draft even documented that fact in a comment directly above
   goldens that lacked it. Caught by the failing test on first run, fixed in both cases.
2. **A `multiedit` on TODO_LIST silently applied 3 of 4 edits.** The failed edit was the
   CI table row — a ~470-char table line I transcribed from an earlier read, and the
   tool reported the partial failure only in its summary. Recovery: full-file rewrite.
   Lesson: for very long table rows, rewrite the file; do not transcribe.
3. **Three false starts on the quality gate from ambient-env leakage.** This session's
   shell exports `GOCACHE`/`GOMODCACHE`/`GOLANGCI_LINT_CACHE` under `/mnt/buildcache`,
   which does not exist here. `verify.sh` "works with or without direnv" — except the
   caches leak through `nix develop -c` too. Fixed by exporting all three to writable
   paths. Not a repo bug (the user's real direnv'd shell is fine), but sessions in
   stripped-down environments hit it immediately.

## e) WHAT WE SHOULD IMPROVE

| IMP | Improvement | Priority |
|-----|-------------|----------|
| IMP1 | **Generate-then-review goldens, never hand-transcribe them.** The scratch-dumper workflow (dump actual bytes → review each line against the protocol → pin) caught nothing this time only because the test run caught my omission instead. Keep the review step mandatory: a golden pinned blind is circular. | Medium |
| IMP2 | **Check the daemon's actual state before trusting a TODO's premise.** The datastartest "sits uncommitted" item was a session old and already stale. `git -C <repo> log/status` is one command; a TODO premise is not evidence. | Medium |

## f) Up to 50 things we should get done next

1. Add `actionlint` to go-sse CI and/or the devShell (TODO_LIST CI & tooling; go-datastar precedent `5887043`).
2. Decide browser-E2E scope (Option B vs C) to unblock the chromedp test (TODO_LIST Blocked).
3. Decide `38e79aa` amend-vs-accept (TODO_LIST Blocked; the AGENTS.md Gotcha is the cheap compromise either way).
4. Cut the next ssetest release, then the paired datastartest tag (TODO_LIST Cross-repo, BLOCKED on exactly this).

## g) QUESTIONS (cannot answer myself)

None open — the three decision-gated items are already phrased as user decisions in TODO_LIST.
