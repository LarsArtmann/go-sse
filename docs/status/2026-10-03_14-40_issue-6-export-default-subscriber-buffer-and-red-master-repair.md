# Status Report — 2026-10-03 14:40 — Issue #6: Export `DefaultSubscriberBuffer` + Red-Master Format-Gate Repair

Single-issue session: resolved
[LarsArtmann/go-sse#6](https://github.com/LarsArtmann/go-sse/issues/6) (export the
per-subscriber default buffer 64 as a constant so downstream hubs match by
construction). Along the way, probed CI per standing rule, found master red on
the `Nix flake check` job (treefmt), diagnosed it as the daemon-committed
non-canonical script formatting, and repaired it. Final state: two surgical
local commits (`fb47cb0` feat/Resolves #6, `6e7e80e` fix(format)), working tree
clean, **not pushed** (no request). Full gate (`scripts/verify.sh`): ALL CHECKS
PASSED — treefmt, GOEXPERIMENT guard, tidy + directive alignment, vet, lint 0
issues ×3 modules, race tests green on all 4 module trees, bench smoke, `nix
flake check`. Honest caveat in b)2: the full gate ran on a near-final tree;
the last two comment-only edits were re-verified piecemeal (build ×3 modules,
root race tests, `nix fmt` 0-change), not with a fresh full gate.

- cover: library 99.3% (=), ssetest 100.0% (=), sseparse 99.2% (=)

## TL;DR

- Issue #6 done in code and docs: `DefaultSubscriberBuffer = 64` exported at
  `fanout.go:19` (direct rename, no alias — one source of truth), every doc
  reference and the `Health()`-based test pins now derive from the constant.
- Master was red before this session started (`Nix flake check` → treefmt
  build failure on commit `59d5173`, a daemon "heuristic" sweep). Fixed by
  re-canonicalizing 4 shell scripts through `nix fmt` (indentation-only diff).
- The auto-commit daemon swept this session's work into two meaningless
  "chore: auto-commit N changed file(s)" commits; I soft-reset both (local
  only, content preserved) and recommitted surgically with real messages.
- Nothing is pushed. "Resolves #6" only closes the issue once the commit
  reaches the default branch — the issue is still open on GitHub right now.

## a) FULLY DONE

| #   | Work                                                                                                                                                                                                                   | Evidence                                                                                                                                                                                                                                                              |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `defaultSubscriberBuffer` → exported `DefaultSubscriberBuffer` (direct rename, no alias); doc comment rewritten to exported-const style (includes the `[WithBufferSize]` fallback rule)                                 | `fanout.go:14-19`; all 5 internal references updated (`fanout.go:32,98,128,149` + doc prose); `GOWORK=off go vet` clean                                                                                                                                               |
| 2   | Default-buffer test pins derive from the export instead of the literal 64 (exactly the "desired" column of the issue's table)                                                                                           | `lifecycle_test.go:210-215,298-303,308-313` (`TestBroadcaster_Health_InitialState`, `_NonPositiveIsIgnored` sites); `go test . -race -count=1` green                                                                                                                  |
| 3   | All godoc references point at the constant (docs stay true if the value ever changes)                                                                                                                                  | `fanout.go` (`WithBufferSize`, `BroadcasterHealth.BufferSize`, `Subscribe` docs), `broadcaster.go` (`Broadcast` drop-policy doc), `example_test.go` (`ExampleBroadcaster`)                                                                                            |
| 4   | End-user + project docs updated: README drop policy and design decision now name the constant and its tunability; FEATURES inventory row; AGENTS.md gotcha entries                                                     | `README.md` (2 sites), `FEATURES.md` (WithBufferSize row), `AGENTS.md:97,109`                                                                                                                                                                                         |
| 5   | Red-master repair: 4 shell scripts re-canonicalized through `nix fmt` (shfmt tab→2-space, indentation-only, zero semantic change); this is precisely what the failing CI job wanted                                     | `scripts/verify.sh`, `scripts/golangci-pin.sh`, `scripts/release-verify.sh`, `scripts/smoke-examples.sh`; CI evidence: run `37104833143` failed `Nix flake check` (treefmt drv); post-fix `nix flake check` passes locally                                            |
| 6   | Full verification gate green: `scripts/verify.sh` → ALL CHECKS PASSED (fmt, tidy+directives, vet, golangci-lint 0 issues ×3, race tests root+ssetest+sseparse+examples, bench smoke, `nix flake check` incl. vendor hashes) | gate log (background shell `011`), "all checks passed!" from `nix flake check`                                                                                                                                                                                        |
| 7   | Surgical history restoration: soft-reset the two daemon heuristic commits (content preserved — both were unpushed local-only) and recommitted as two properly-messaged commits                                          | `fb47cb0` "feat(broadcaster): export DefaultSubscriberBuffer constant" (7 files, `Resolves #6`) + `6e7e80e` "fix(format): re-canonicalize shell scripts through nix fmt" (4 files); replaced `59184a8`/`c00cdea` "chore: auto-commit N changed file(s) (heuristic)" |
| 8   | Final-tree sanity: all three modules build; treefmt canonical (0 changed); working tree clean; coverage measured fresh (line above)                                                                                     | `go build ./...` + `(cd ssetest/sseparse && go build ./...)` → BUILD-OK; `nix fmt` → "formatted 0 files (0 changed)"; `git status` clean                                                                                                                               |

## b) PARTIALLY DONE

1. **Issue #6 resolution.** Done: constant exported, references migrated, tests
   derive from the export, commits carry `Resolves #6`. Remaining: **push** —
   until `fb47cb0` reaches master, GitHub keeps the issue open and CI has not
   validated the tree. Also remaining downstream: the issue names go-aichat's
   `ssehub` duplicate (`defaultBufferSize = 64` + lockstep test) as the
   consumer that should be retired; that migration lives in the go-aichat repo
   and has not started (and only becomes possible via the proxy after a tag).
2. **Gate-vs-final-tree integrity.** The full `scripts/verify.sh` gate ran
   green, but I kept editing (`broadcaster.go`, `example_test.go` comment
   refs) *while* the background gate was reading the tree. Those last edits are
   comment-only and were re-verified (vet, root race tests, `nix fmt` 0-change,
   all-module builds), but the honest statement is: the complete gate as a
   single atomic run never saw the exact final tree. Remaining: one clean
   full-gate (or push and let CI be the atomically-clean run).

## c) NOT STARTED

1. **Push master** (ahead by 2) — explicitly not done: no user request, and the
   harness forbids unrequested pushes.
2. **`CHANGELOG.md` [Unreleased] entry** for the new exported API constant —
   forgot it entirely this session (repo convention: completed work goes to
   CHANGELOG). Not even started.
3. **go-aichat `ssehub` migration** to `sse.DefaultSubscriberBuffer` (delete
   local literal + lockstep test) — different repo, needs a tagged release to
   consume cleanly; not started.
4. **Harvest of section f** into `TODO_LIST.md` / `ROADMAP.md` — pending per
   the user's "then wait for instructions" gate; flagged, not executed.

## d) TOTALLY FUCKED UP

1. **I mutated the source tree while the verification gate was running.**
   Started `scripts/verify.sh` in the background, then edited
   `broadcaster.go`/`example_test.go` before it finished. The gate's green
   verdict attached to a tree that no longer existed by the time it printed.
   Catching the last two comment edits was routine this time; the *pattern*
   is how a real regression sails through as "gate green". Tree must freeze
   during gates.
2. **I ignored the repo's own loudest lesson about the commit daemon.**
   `AGENTS.md` says: "Commit surgically and immediately when the message
   matters" — and this change matters (public API, resolves a filed issue). I
   batched all edits first, so the daemon swept 12 files into two
   content-blind "chore: auto-commit N changed file(s)" commits and I had to
   soft-reset local history to fix the messages. The rewrite was safe
   (unpushed) but entirely avoidable churn with real risk attached.
3. **`Resolves #6` in an unpushed commit closes nothing.** I narrated the
   session as "resolved" while the issue remained open on GitHub and CI blind.
   Resolution is a property of the pushed default branch, not of a local
   commit message.
4. **CHANGELOG.md never touched.** A new exported identifier is a
   user-facing addition; the repo convention is explicit ("Completed work goes
   to `CHANGELOG.md` [Unreleased]"). Skipped it and only noticed writing this
   report.
5. **First format check was a clumsy compound** (`nix fmt -- --check || nix
   build .#checks...format`) whose truncated, rerun-looking output obscured
   the actual treefmt failure and cost a round trip. `nix fmt && git diff
   --exit-code` is the whole trick.
6. **`sed -i` on `AGENTS.md`** instead of the edit tool. Worked (I grepped the
   exact lines first), but exact-match editing is the discipline that
   prevents silent collateral damage in tracked files; I used the weaker tool
   for speed on the one file where precision matters most.

## e) WHAT WE SHOULD IMPROVE

| IMP   | Improvement (process, not product)                                                                                                                                                             | Priority |
| ----- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| IMP1  | Freeze the tree during gates: no edits between launching `scripts/verify.sh` (or CI) and reading its verdict. If an edit is truly needed, cancel, edit, relaunch.                              | high     |
| IMP2  | Commit the core API change immediately after its tests go green, before doc polish — this is the concrete defense against the daemon sweep that `AGENTS.md` already prescribes.                 | high     |
| IMP3  | Treat "pushed to default branch" as the definition of issue-resolution; track "issue close pending push" as an explicit TODO item whenever a commit says `Resolves #N`.                        | high     |
| IMP4  | CHANGELOG entry is part of the definition of done for any exported-symbol change — add it to the same commit, not as an afterthought.                                                          | med      |
| IMP5  | Canonical format check is one command: `nix fmt` followed by `git diff --exit-code` (or just let the gate's treefmt step do it); stop composing fragile fallback chains.                        | med      |
| IMP6  | Prefer the edit tool over `sed -i` for tracked files, always.                                                                                                                                  | low      |
| IMP7  | Consider a verify.sh tripwire: snapshot `git status --porcelain` at start, fail loudly if the tree changed mid-gate ("results invalid: tree moved under the gate"). Directly mechanizes IMP1.   | med      |
| IMP8  | Daemon hygiene: heuristic sweeps that mix API changes with format fixes produce exactly the history we had to rewrite; when a session spans multiple logical changes, commit each before starting the next. | med      |

## f) Up to 50 things we should get done next

Grouped by priority; this section is the HARVEST source for
`TODO_LIST.md` (bounded, short-term) and `ROADMAP.md` (long-term).

**P0 — ship this session (blocking, tiny):**

1. Push master (2 commits) so CI validates the final tree.
2. After push: verify all CI jobs green (Test, Lint, Vet, Coverage, Coverage
   gate, Examples, Smoke, Shellcheck, Nix flake check, Vulncheck, Fuzz).
3. Confirm issue #6 auto-closed by `fb47cb0`; comment the commit hash on the
   issue if it doesn't.
4. Add `CHANGELOG.md` [Unreleased] entry: "Added: `DefaultSubscriberBuffer`
   exported constant (resolves #6)".
5. Run HARVEST: move bounded f-items into `TODO_LIST.md`, long-term ones into
   `ROADMAP.md`, and delete anything already done.

**P1 — make the export maximally useful (needs a release or cross-repo work):**

6. Cut the next release (additive exported const → minor bump, e.g. v0.7.0)
   so downstream consumes the constant from the module proxy, not a replace.
7. Migrate go-aichat `ssehub` to `sse.DefaultSubscriberBuffer`; delete its
   local `defaultBufferSize = 64` literal and the lockstep spec test.
8. Verify go-aichat's CI is green after the migration (lockstep test removal
   must not leave a coverage hole behind).
9. Add a `go-aichat` compat workflow analogous to `datastar-compat.yml`
   (weekly `go get go-sse@latest` + test) so downstream breakage surfaces here.
10. Update the fan-out guide in `docs/guides/` to reference the constant
    wherever it discusses buffer sizing (check first whether it mentions 64).
11. README API-surface section: add `DefaultSubscriberBuffer` next to
    `WithBufferSize` so the constant is discoverable where options are listed.
12. Add `ExampleWithBufferSize` (or extend `ExampleBroadcaster` prose) in
    `example_test.go` demonstrating `DefaultSubscriberBuffer` vs tuned size.

**P2 — process/tooling hardening (from this session's mistakes):**

13. Implement the verify.sh tree-changed tripwire (IMP7).
14. Decide a standing policy for daemon commits in API-change sessions: commit
    before the daemon's sweep window, or configure the daemon to skip `.go`
    files with uncommitted session edits (hook candidate).
15. Record the "mutating tree under a running gate" failure mode in
    `references/lessons.md` (cross-project: any gate + any edit loop).
16. Add "Resolves #N ⇒ must push" to the pre-finish checklist (Crush config or
    AGENTS.md pre-push gate section).
17. CHANGELOG discipline: a one-line template entry in the commit message
    convention ("+changelog: ...") so future sweeps can't lose it.

**P3 — known repo threads (noted during the session, no new research):**

18. Re-run `nix flake check --all-systems` before the next release (aarch64
    targets are omitted by declaration; release-time check is the contract).
19. Keep the golangci-lint pin aligned through the next flake update
    (`scripts/golangci-pin.sh` cross-check output → bump ci.yml in that PR).
20. Watch for the buildflow-side durable fix: normalize target should derive
    from max(dep directives) instead of the hardcoded `go 1.27` re-lowering
    (three re-fixes logged 2026-10-01; verify.sh's directive gate is the
    detector, the fix lives in buildflow).
21. Standing control (policy decision) for Actions-created PRs held in
    `action_required` until human approval — approve per session / workflow_run
    self-gating / keep manual; still tracked in TODO_LIST.
22. Re-add `aarch64-darwin`/`aarch64-linux` to flake `systems` when nixpkgs
    support returns (declared omission, revisit on flake updates).
23. Consider annotating the rewritten-away daemon commits (`59184a8`/
    `c00cdea`) — they no longer exist in history after the soft-reset, so the
    only trace is this report; nothing to do unless the daemon re-offends.
24. Next status report should confirm the "gate on final tree" gap (b)2) is
    closed by an atomic full-gate or CI run.
25. Sweep `docs/` (non-archived) for any remaining prose that states the
    buffer default as a bare unexplained "64" and point it at the constant
    (README is done; guides unknown — check, don't assume).
26. `gopls infertypeargs` residual infos in `example_test.go:120,122` are
    documented WONT — re-verify they stay editor-only after the next gopls
    update (`gopls check` CLI reports 0).

(26 items — stopping where the list stops being honest padding. Items 18–22
are standing repo threads restated here so HARVEST finds them in one place.)

## g) Questions I CANNOT figure out myself

1. **Push now or do you want to review first?** Master is +2 (`fb47cb0`,
   `6e7e80e`); origin/master is red on the format gate until the push lands,
   and issue #6 stays open. I will not push without your explicit go-ahead —
   say "push" and I'll do it and babysit CI.
2. **Release sequencing for the downstream migration:** do you want a minor
   release (e.g. v0.7.0) tagged *before* I migrate go-aichat's `ssehub` to the
   constant (proxy-consumable, no local replace), or should the go-aichat
   migration wait for whatever ships next anyway?
3. **Daemon-commit policy going forward:** when a session's work matters
   enough to deserve a real message (like this one), do you want me to keep
   soft-resetting unpushed daemon heuristic commits and recomitting surgically
   (what I did today), or leave daemon commits untouched and only add
   well-messaged commits on top? Today's rewrite was safe because nothing was
   pushed — I'd like a standing rule before the next time.
