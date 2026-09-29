# Status Report — 2026-09-29 07:03 — Post-Release CI Health, VendorHash Repair, and PR Closure (go-sse + go-datastar)

Session scope: executed the handoff from the sseparse-release session. Found
and fixed TWO distinct go-datastar CI reds (lint-pin panic + a hidden stale
vendorHash that nix.yml's path filter had silently skipped), un-red go-sse
master (which was ALSO red — mid-flight work from the concurrent session),
adopted both go-sse dependabot bumps on master, closed all three open PRs,
annotated the go-daemon adoption feedback doc, verified all releases live on
tags/GitHub/proxy, and documented the new `go 1.27.1` directive floor.
Gates at close: go-sse `scripts/verify.sh --fast` ALL CHECKS PASSED +
`nix flake check` all checks passed + master CI green (runs 36516730859,
36523054210); go-datastar `nix flake check` all checks passed + master CI
and nix jobs green (runs 36516328568/36516328606, then 36523182096/36523182190
on the follow-up dep sweep).

- cover: library 99.3% (=), ssetest 100.0% (+2.0), sseparse 99.2% (=)
- cover (go-datastar): root 98.8% (new), datastartest 95.5% (new), broadcast 87.8% (new)

## TL;DR

- go-datastar master was double-red: golangci-lint v2.12.2 panics in
  go-tools' buildir pass in CI (devshell's 2.13.2 is clean), AND the nix gate
  was silently broken since c1826fb (stale datastartestVendorHash + a path
  filter that never triggered on `datastartest/go.mod` edits).
- go-sse master was red too (bodyclose + treefmt on the concurrent session's
  mid-flight work); their two unpushed commits fixed it — verified and pushed.
- All three open go-sse PRs closed: #3/#5 satisfied on master (bumps adopted
  with re-derived vendor hashes), #4 superseded (master's flake.lock is
  newer) — and #4's CI had NEVER run (GitHub's workflow-approval gate for
  Actions-created PRs).
- The ssetest go-sse v0.6.1 bump exposed a HARD directive floor: go-sse
  v0.6.1 on the proxy declares `go 1.27.1`; all three go.mod files now
  aligned at 1.27.1 with the rationale recorded in AGENTS.md.

## a) FULLY DONE

| #   | Work                                                                                                                                                                                                                                    | Evidence                                                                                                                                                                                                                          |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | go-datastar lint job un-red: CI pin v2.12.2 → v2.13.2 (buildir panic `unexpected expr: *ast.KeyValueExpr`, exit 3, on dependency package `poll`), restoring CI==devshell parity in ci.yml, flake `lint-ci` app, CONTRIBUTING, AGENTS.md | `nix run .#lint-ci` → 0 issues; CI run 36511560281 (fail, panic log) vs 36516328568 (green, 1m39s); commit `98e465c`                                                                                                                |
| 2   | go-datastar nix gate repair: `datastartestVendorHash` re-derived (replace-drop was NOT hash-inert), `nix.yml` path filter widened root-only `go.mod`/`go.sum` → `**/go.mod` + `**/go.sum`                                                | hash `sha256-QAoGudgZ2+…` (got: from FOD failure); local `nix flake check` all checks passed; nix workflow run 36516328606 green; the filter gap proven by c1826fb never triggering nix.yml                                          |
| 3   | go-datastar docs: CHANGELOG `[Unreleased]` Fixed entries (lint pin + vendorHash/filter); AGENTS.md gotcha "inert replace drops are NOT vendorHash-inert; the nix CI path filter can hide it"                                             | CHANGELOG.md lines 10–27; AGENTS.md new bullet before the modRoot entry                                                                                                                                                             |
| 4   | go-sse master un-red: verified and pushed the concurrent session's 2 unpushed commits (corpus generator + scripts/shfmt/flake polish) that fix bodyclose + treefmt failures                                                             | local ssetest lint 0 issues; `nix run .#test-race` all 6 modules ok; `nix flake check` green; push `24134ad..2ab51ef`; CI run 36516730859 green (7m51s)                                                                             |
| 5   | go-sse dependency bumps adopted on master: ssetest go-sse v0.6.0→v0.6.1 (+ go-branded-id v0.6.0 indirect), root go-error-family v0.10.1→v0.10.2; vendor hashes re-derived                                                                 | `go get` output (upgraded lines); FOD `got:` hashes pasted into flake.nix; `nix build .#checks…build` + `…build-ssetest` exit 0; dependabot auto-closed #3 and #5 ("go_modules … Update SUCCESS" runs 36518180282/36518179794)      |
| 6   | Directive floor established and documented: all three go.mod files at `go 1.27.1` (hard floor: go-sse v0.6.1 on the proxy declares 1.27.1, verified via `go mod download -json` + .mod inspection)                                        | `grep -H "^go " go.mod ssetest/go.mod sseparse/go.mod` → 1.27.1 ×3; `verify.sh --fast` ALL CHECKS PASSED; AGENTS.md GOEXPERIMENT paragraph rewritten; commit `96f6b37`, CI run 36523054210 green                                      |
| 7   | go-sse PR #4 (weekly flake update) closed as superseded with root-cause note: master's flake.lock nixpkgs rev `7a0f122f` (2026-09-28 06:58Z) is NEWER than the PR's `e158d9ed` (2026-09-26); its CI never ran (`action_required` gate)  | `gh api repos/NixOS/nixpkgs/commits/<rev>` dates; `gh pr checks 4` output; close comment on PR #4                                                                                                                                    |
| 8   | go-daemon adoption feedback doc annotated with resolution evidence (tags live, go-daemon `36c2ecc` corpus conformance, pdd cascade, go-datastar lockstep)                                                                                 | `docs/feedback/new/2026-09-29_ssetest-gaps-blocking-go-daemon-adoption.md` header blockquote updated (committed by daemon in `ad9d063`)                                                                                             |
| 9   | All releases verified live end to end: sseparse/v0.1.0 + ssetest/v0.4.0 (tags, GitHub Releases "Latest", version-specific proxy probes) and go-datastar v0.6.1 ×4 (same)                                                                  | `go list -m -json <mod>@<ver>` with GOPROXY=proxy.golang.org → Version hits; `gh release list` on both repos                                                                                                                        |
| 10  | go-datastar follow-up dep sweep (other session's error-family v0.11.0 bump + 3 re-derived hashes) verified locally and pushed                                                                                                            | local `nix flake check` all checks passed BEFORE push; CI 36523182096 + nix 36523182190 green after; push `98e465c..3fc99fe`                                                                                                        |

## b) PARTIALLY DONE

1. **go-datastar tag-CI on v0.6.1** — master is green, but the four frozen
   v0.6.1 tags keep permanently red tag-CI (the workflow file at a tag is
   immutable; the old pin + stale hash live there forever). Shipped: the
   CHANGELOG `[Unreleased]` note documenting this. Remains: the decide-and-
   maybe-execute of a v0.6.2 re-cut (question for Lars, section g). No
   technical blocker — release content was verified green before tagging.
2. **TODO_LIST harvest of this report's section f** — the report is written,
   but nothing from section f has been harvested into TODO_LIST.md /
   ROADMAP.md yet. Deliberately paused: the concurrent session actively edits
   TODO_LIST (11-file sweeps today), and the user instruction was
   "write report, then WAIT". Remaining: HARVEST after coordination.
3. **Concurrent session's 03-45 plan tail (M4–M17)** — observed large chunks
   landing today from the OTHER session (GOEXPERIMENT removal `1c0fc85`,
   CI coverage-gate + shellcheck `41aac66`, ssetest 100% coverage `88189b8`,
   corpus generator `91fe8b7`, templ pin policy `84ecbee`, benchstat
   baselines `01051f2`). Not my ledger — I deliberately did not duplicate
   it — but "the hygiene tail is done" cannot be claimed by this session;
   it can only be claimed by theirs.

## c) NOT STARTED

1. **Workflow-approval gate for Actions-created PRs** (both repos) — the
   weekly flake-update workflow pushes a branch and opens a PR under
   GITHUB_TOKEN; GitHub then holds ALL pull_request CI runs in
   `action_required` until a human approves. Nobody approved PR #4 in 26
   hours, so it merged-blind risk sat there unnoticed. I documented it in
   the close comment but configured nothing (approve bot, workflow_run
   pattern, or repo setting). Blocked on: a policy decision (see g).
2. **Dependabot-on-nix-hash policy** — every dependabot PR that bumps a
   go.mod in a vendorHash-carrying module will now (correctly) fail the nix
   job with a hash mismatch, and dependabot cannot fix nix hashes. Today I
   handled it ad hoc (adopt-the-bump-on-master). No standing policy exists
   (auto-close-and-adopt? maintainer hash-fix push? `nix` job made
   non-blocking for dependabot branches?). Started: no. Blocked on: policy.
3. **Directive-floored normalize** — buildflow's normalize step lowered
   root/sseparse to `go 1.27` twice today, fighting the hard 1.27.1 floor
   imposed by go-sse v0.6.1. The fix (normalize to max(dep directives)
   instead of a hardcoded version) lives in buildflow, not go-sse. Not
   started; flip-flop currently prevented only by my AGENTS.md rationale.

## d) TOTALLY FUCKED UP

1. **Trusted the handoff's "devshell v2.14.0" claim without measuring.**
   go-datastar's devshell actually ships golangci-lint **2.13.2**. Acting on
   the claim would have bumped CI to v2.14.0 and re-created the exact
   pin-skew class go-sse's AGENTS.md documents. Caught only because I
   measured before editing. Handoffs are context, not ground truth.
2. **Started fixing go-datastar CI without first running its gates.** I did
   not run `nix flake check` before editing — I only discovered master had
   been nix-broken since c1826fb (stale datastartestVendorHash) as a
   side-effect of validating my lint fix. Master sat red-and-silent for ~90
   minutes of my session (and unknown time before) because nix.yml's path
   filter skipped the commit. The "verify current state before changing it"
   rule exists precisely for this.
3. **The directive flip-flop: two agents fought over three lines of
   go.mod for ~40 minutes.** I raised root/sseparse to 1.27.1; the other
   session's normalize lowered them back; the daemon committed both states.
   Neither of us announced go.mod-sensitive work. Resolved only by re-raising
   with a loud AGENTS.md rationale the other session cannot miss.
4. **Measured go-datastar coverage with `./datastartest/...` from the root
   module** — silent setup failure (separate module), reported 0.0%, which I
   nearly wrote into this report. Caught because 0.0% is implausible; the
   real numbers (95.5% / 87.8%) required cd-ing into each module.
5. **A botched `cd X && gh …` composite call** produced go-sse output that I
   briefly read as go-datastar's, momentarily concluding my go-datastar fix
   had failed. Separate tool calls per repo from then on; should have been
   that way from call one.
6. **Dependency-bump commit landed inside a daemon sweep titled "check in
   benchmark baselines"** (`01051f2`) — my go.mod/vendor changes carry a
   message that has nothing to do with them. AGENTS.md already documents the
   plausible-but-false daemon-message class; I added two more instances to
   master history instead of committing surgically ahead of the sweep.

## e) WHAT WE SHOULD IMPROVE

| IMP   | Improvement                                                                                                                                                                                                                                               | Priority |
| ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| IMP1  | Make `nix flake check` a mandatory pre-push step for ANY push touching flake.nix or ANY module's go.mod/go.sum in every repo — path filters provably cannot be trusted to gate this (two silent master breakages today).                                       | high     |
| IMP2  | Session-start CI health probe: `gh run list --branch master --limit 5` in every repo a session touches, BEFORE trusting any local tree state. go-sse master was red for two runs before anyone looked.                                                           | high     |
| IMP3  | Coordinate go.mod/directive/vendorHash-sensitive edits between concurrent sessions (claim-and-announce, or a lockfile note in a shared scratch file). The directive flip-flop wasted ~40 minutes across two sessions.                                              | high     |
| IMP4  | Verify handoff facts empirically before acting on them (versions, hashes, paths). The v2.14.0-vs-2.13.2 error came from an unverified handoff line.                                                                                                             | high     |
| IMP5  | Commit release-adjacent changes surgically and immediately; the 60-second daemon sweep cadence guarantees wrong messages for anything left dirty (today: dep bumps inside "benchmark baselines").                                                                 | med      |
| IMP6  | Give Actions-created PRs a self-gating story: either auto-approve workflow runs for the bot branch (repo setting/API), switch the flake-update workflow to `workflow_run`-triggered checks, or stop opening PRs that cannot run CI.                                | med      |
| IMP7  | Fix the stale `flake.nix` comment in go-datastar (lines ~49–53 still claim datastartestVendorHash moves on ANY tracked repo-root edit — contradicted by the ADR-004 minimal filesets; verified today: root .md/.yml edits do NOT move it).                         | med      |
| IMP8  | Cross-repo coverage measurement needs a per-module recipe (separate modules need cd-into-module); a one-liner script per repo would have prevented the 0.0% mis-measure.                                                                                          | low      |

## f) Up to 50 things we should get done next

Grouped by priority; impact/effort noted. This is HARVEST fodder for
TODO_LIST.md (bounded items) and ROADMAP.md (long-term), after verification
against code — several will be done by the concurrent session meanwhile.

### Critical

1. Decide and execute the go-datastar v0.6.2 question (re-cut to get green
   tag-CI, or accept frozen red tag-CI on v0.6.1). User decision; execution
   ~30 min if yes. Impact: Critical (release hygiene). Effort: S.
2. Handle the `action_required` approval gate for Actions-created PRs on
   go-sse and go-datastar (approve current/future flake-update runs; or
   switch the workflow to a self-gating pattern). Impact: Critical — weekly
   automation currently produces PRs nobody can trust. Effort: S–M.
3. Fix buildflow's normalize step to derive the target go directive from
   max(dep directives) instead of a hardcoded value, ending the 1.27/1.27.1
   flip-flop at the root. Impact: Critical (recurring gate churn). Effort: M.
4. go-datastar: fix the stale vendorHash-movement comment in flake.nix
   (~lines 49–57) to match the ADR-004 minimal filesets, so the next
   session does not fear ghost hash movement. Impact: High. Effort: S.

### High

5. Audit both repos for OTHER master commits that path filters silently
   skipped (join `git log --name-only` against every workflow's `paths`) —
   today's nix.yml gap was found by luck, not by process. Effort: M.
6. Add `nix flake check` (or `--fast` verify) to the pre-push habit docs of
   go-datastar CONTRIBUTING the way go-sse documents verify.sh. Effort: S.
7. go-datastar: enforce the golangci pin mechanically — a verify-style
   cross-check script (go-sse's `scripts/golangci-pin.sh` pattern) so
   CI==devshell skew fails locally before push. Effort: S.
8. Raise coverage-gate floors in go-sse now that ssetest is at 100.0%
   (ssetest floor 95 → 100; consider library 90 → 99, sseparse 95 → 99) to
   make the new levels the enforced minimums. Effort: S.
9. go-sse CHANGELOG `[Unreleased]`: add entries for the dep bumps
   (go-sse v0.6.1 in ssetest, error-family v0.10.2) + the 1.27.1 directive
   alignment — the daemon commits shipped none. Effort: S.
10. go-datastar CHANGELOG `[Unreleased]`: add the error-family v0.11.0 bump
    entry (their 11-file sweep touched go.mod ×3 but not CHANGELOG).
    Effort: S.
11. Bump pdd's go-sse pin v0.6.0 → v0.6.1 (+ vendorHash) so the consumer
    chain is uniformly on v0.6.1; go-daemon likewise. Effort: S–M each.
12. Verify tag-CI state for go-sse's sseparse/v0.1.0 + ssetest/v0.4.0 tags
    (same frozen-tag-CI class as go-datastar; unknown whether they ran
    green at tag time). Effort: S.
13. Sweep go-datastar's open dependabot PRs under the new reality (nix job
    will fail on hash moves; adopt-or-close each deliberately). Effort: M.
14. Proactive go-error-family v0.11.0 evaluation for go-sse root (go-datastar
    already bumped; a dependabot PR will arrive anyway — get ahead of it with
    the vendor-hash dance done once). Effort: S–M.
15. Update go-sse AGENTS.md "go-directive claims" one more time if/when the
    next root tag ships at 1.27.1 (the tag-vs-master directive split is now
    v0.6.1@1.27.1 vs master@1.27.1 — re-check on next release). Effort: S.

### Medium

16. Record the cross-project lesson in crush-config
    `references/lessons.md`: "verify handoff facts + measure tool versions
    before pin changes; path filters are not gates". Effort: S.
17. Run a local 5-minute fuzz soak on go-sse's bumped tree (deps changed
    today; CI's scheduled Fuzz runs green but a fresh soak is cheap).
    Effort: S.
18. Confirm go-sse's weekly `datastar-compat.yml` workflow ran/passed after
    go-datastar v0.6.1 (consumer compat gate). Effort: S.
19. Annotate `docs/status/2026-09-29_01-58_sseparse-release-and-consumer-chain.md`
    with resolution markers (tags live) per the ANNOTATE convention, then
    archive when fully resolved. Effort: S.
20. Check go-datastar's private-vulnerability-reporting repo setting (was
    enabled for go-sse last session; go-datastar state unverified). Effort: S.
21. Verify go-sse's coverage-gate measures root as package `.` (the doc's
    scope rule) — if it uses `./...` the gate understates by ~40 points.
    Effort: S.
22. go-datastar: wire a coverage gate for datastartest (95.5%) and broadcast
    (87.8%) if their coverage workflow lacks thresholds. Effort: S–M.
23. Document the "Actions-created PR approval gate" behavior in both repos'
    AGENTS.md (currently only in a PR close comment — invisible to future
    sessions). Effort: S.
24. go-daemon: re-run its corpus conformance after the next sseparse tag to
    keep the consumer oracle chain warm. Effort: S.
25. Harvest section f into TODO_LIST.md / ROADMAP.md (docs-health HARVEST),
    after coordinating with the concurrent session that owns TODO_LIST
    today. Effort: M.

### Low / Roadmap fuel

26. Automate module-proxy warm-up (`@v/list` staleness purge + version
    download) into `scripts/release-verify.sh` for future first-tags.
27. Add a `smoke` flake app caller to go-datastar CI if it gains runnable
    examples (go-sse has the job; go-datastar's app exists — parity check).
28. Evaluate a dependabot config change: group all go.mod bumps per module
    so one PR = one hash-fix instead of three.
29. Add `broadcastVendorHash`/`datastartestVendorHash` derivation notes to
    go-datastar CONTRIBUTING (the paste-loop recipe, ADR-004 references).
30. Consider `GOTOOLCHAIN=auto` pinning policy documentation (devshell pins
    the toolchain; ambient 1.26 gopls false positives still confuse — keep
    the "trust CLI over editor" note current).
31. go-datastar: extend the JS-version drift test to broadcast/static docs
    mention in CHANGELOG (JS-version-in-CHANGELOG test exists for root).
32. Split the go-datastar CI lint job per module for clearer failures (one
    panic currently fails the whole matrix).
33. Add a scheduled `nix flake check --all-systems`-equivalent note for
    aarch64-darwin when nixpkgs support returns (both repos carry the
    systems pin).
34. go-sse: cover the `Stream.Send` short-write path in an example test if
    not already (doc claims pinned — verify the pinning test exists).
35. Keep `docs/status/` archive hygiene: audit for all-resolved reports
    older than the 2026-09-29 batch and `git mv` to `archived/`.
36. Evaluate a shared "session handoff facts" template that separates
    MEASURED facts from CLAIMED facts (today's v2.14.0 error).
37. go-datastar: pin golangci-lint version in ONE place (flake app) and
    have ci.yml read it (action input) to eliminate the 4-site pin.
38. Consider Renovate vs Dependabot for the nix-hash-aware update flow
    (Renovate supports lock-file maintenance hooks better).
39. Add release-verify.sh probes for the OTHER nested module (ssetest case)
    if only the sseparse case was added last session — parity audit.
40. go-sse: module-boundary test for "root must never require ssetest or
    sseparse" — confirm it also covers indirect requires via go.sum.
41. Track go-branded-id v0.6.0 in go-sse's own root go.mod (ssetest got it
    transitively; root still pins its own version — align on next bump).
42. Add a `directive-equality` check to go-datastar's gates (go-sse has one
    in verify.sh; go-datastar has 3 modules too).
43. Document in go-sse AGENTS.md that `verify.sh` skips the flake check
    under `--fast` and MUST be paired with a flake check for go.mod/flake
    pushes (trap hit today at go-datastar scale).
44. Evaluate caching golangci-lint 2.13.2 binary in CI cache key bump policy
    (cache key now v2.13.2 — remember to bump BOTH key and install line next
    time; note it in the workflow comment).
45. go-datastar AGENTS.md: add the "CI health probe at session start" step
    to the commands section (IMP2 institutionalized).
46. Sweep stale `nolint`/exclusion comments in ssetest `.golangci.yml` that
    referenced the pre-split layout (bodyclose exclusion for collect.go —
    still correct, but re-verify after the 100% coverage work).
47. Consider tagging go-sse root v0.6.2 only after directives settle a
    release cycle at 1.27.1 (avoid churning the proxy).
48. Add PR template checkbox for "ran nix flake check" in both repos.
49. Investigate why PR #4's branch ALSO carried a stale ci.yml pin (v2.13.2
    at branch point) — confirms skew existed on master pre-2026-09-28; write
    the timeline into the pin-parity AGENTS.md note.
50. Celebrate: ssetest hit 100.0% statement coverage today (other session)
    — update FEATURES.md if it tracks coverage milestones.

## g) Questions I CANNOT figure out myself

1. **go-datastar v0.6.2: re-cut or not?** The four v0.6.1 tags have frozen
   red tag-CI (immutable old workflow: panicking lint pin + stale hash).
   Release content was verified green pre-tag and master is green. Do you
   want a v0.6.2 re-cut purely for green tag-CI, or is master-green +
   the CHANGELOG note the accepted end state? I tried to answer via the
   repos' release conventions (ADR 002 lockstep, CONTRIBUTING) — neither
   speaks to re-cutting for CI cosmetics.
2. **What is the intended control for Actions-created PRs (weekly flake
   update)?** Their CI sits in `action_required` until a human approves —
   so the weekly automation currently opens PRs that can never gate
   themselves. Should I (a) approve runs routinely as part of sessions,
   (b) restructure the workflow to self-verify before opening the PR (run
   verify.sh inside the workflow, PR becomes a formality), or (c) is
   manual approval the deliberate safety control you want kept?
3. **How should concurrent sessions coordinate go.mod/directive/normalize
   work?** Today two sessions flip-flopped the same three go.mod lines.
   Do you want a convention (claim file in a shared note before editing),
   is one session per repo the rule and today's overlap was the anomaly,
   or should buildflow's normalize simply be fixed to respect the
   dependency-derived floor (my recommendation, task #3)?
