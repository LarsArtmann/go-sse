# Status Report — 2026-10-03 15:36 — BuildFlow Gate Green: Flip-War Root Cause, Findings Triage, Doc-Link Repair

**Session scope:** Started from a failing `buildflow --fix --build-mode=full` run (exit 69, 4 failed
steps, 6 finding classes). Session goal: make the gate actually pass — not silence it — and root-cause
why the same two artifacts kept "repairing" themselves backwards all day (a duplicate
`eventBrand.Name` method in `event.go` re-injected five times; `scripts/*.sh` tab-re-indented three
times).

**Final gate state:**

- `scripts/verify.sh`: **ALL CHECKS PASSED** (fmt + vet + lint + test + bench smoke + tidy/directive
  gates + `nix flake check`, run twice after the final tree shape)
- `buildflow --fix --build-mode=full --budget 5m`: **EXIT=0** (was exit 69 at session start)
- `nix flake check`: green (inside verify.sh; hermetic builds for root + ssetest + sseparse)
- CI on the pushed tip (`1b571a8`): **success** (run 37126765073, ~7 min, includes Coverage gate);
  previous CI push (12:46 UTC) was green; actionlint on the tip green

- cover: library 99.3% (=), ssetest 100.0% (+2.0), sseparse 99.2% (=)

## TL;DR

- The "duplicate `eventBrand.Name` method" that reddened the tree five times today was **not** a
  stale editor buffer and **not** (primarily) the commit daemon — it was **BuildFlow's own
  brandid-lint repair stub**. The linter only recognizes a `Name()` method whose body returns a
  string *literal*; our constant-returning method was flagged BD001 on every run, and every
  `buildflow --fix` inserted `func (eventBrand) Name() string { return "event" }` right next to the
  real method — a guaranteed compile error. Fixed at the root by returning the literal `"SSEEvent"`
  (`30ce48b`); brandid-lint now reports 0 findings and has nothing to re-insert.
- The script-format flip-flop was the same disease, different organ: buildflow's bare `shfmt -w`
  (tab indent) against treefmt's canonical 2-space. Skipped via `.buildflow.yml` with rationale
  (`30ce48b`), after an appended-duplicate-YAML-key mistake briefly un-skipped
  go-structure-linter (`bb3226a` fixes that).
- Cleared every auto-fixable finding class: flake meta (homepage/mainProgram/platforms on all three
  check packages — after learning `mainProgram = null` is rejected by `nix flake check`), statix
  repeated-keys, the v31 → v31.11.1 action-comment drift (the pinned SHA was already v31.11.1 —
  verified against the GitHub API before touching anything), and all 8 lychee broken links.
- Triaged and **declined with rationale** the recurring advisory findings (samber/lo suggestions,
  encoding/json v2 migration notes, vendorHash extraction) — they conflict with this repo's
  deliberate zero-dependency and documented-convention policies.
- Verified end-to-end twice: `scripts/verify.sh` ALL CHECKS PASSED, full buildflow run EXIT=0.

## a) FULLY DONE

| #   | Work                                                                                                                                                                                                                                                                                    | Evidence                                                                                                                                                                                                                                                                                    |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Duplicate `eventBrand.Name` compile error removed **and root-caused**: brandid-lint's `parseNameReturnValue` only accepts a `return "literal"` body, so the constant-returning method was flagged BD001 and buildflow's repair re-inserted its stub next to it on every run | `30ce48b` (method returns literal `"SSEEvent"`, const dropped, test comment updated); removal commits `75476e6`, `6a62140`, `346fd2c`; linter source read at `BuildFlow/vendor/github.com/larsartmann/go-branded-id/linter/brands.go` (`collectNameMethods` → `parseNameReturnValue`); `buildflow -s brandid-lint --format finding` → 0 findings |
| 2   | Script tab-flip root-caused and neutralized: buildflow's `NewShfmtProvider` runs bare `shfmt -w` (default tab indent), which rewrote the treefmt-canonical 2-space scripts on every run (proved with a /tmp repro: `shfmt -d` rejects 2-space, `-w` writes tabs)             | `30ce48b` adds `shfmt` to `skip_steps` in `.buildflow.yml` with full rationale; `nix fmt` re-canonicalized and `ebbc28b` committed the scripts; scripts stayed clean through the final full buildflow run                                                                                                    |
| 3   | Flake meta completion on all three hermetic check packages (`hermeticCheck`, `hermeticCheckSsetest`, `hermeticCheckSseparse`): added `homepage`, `mainProgram`, `platforms`                                                                                                  | `3659c65` + `626272b` in `flake.nix`; `mainProgram = null` was tried first and **rejected by nix** (`NIX_MAIN_PROGRAM` env attribute can't be null — caught by verify.sh's flake check), replaced with pname values; `buildflow -s flake-meta-checker` → 0 findings                                            |
| 4   | statix "repeated keys" cleared: four `checks.<name> =` assignments grouped into one `checks = { … }` attrset                                                                                                                                                                 | `3659c65` in `flake.nix:194-199`; `statix check .` → 0 findings                                                                                                                                                                                                                                              |
| 5   | github-actions-pinning findings cleared: the pinned SHA `13d8dd58…` was verified to **already be** v31.11.1's tag SHA (via `gh api repos/cachix/install-nix-action/git/ref/tags/v31.11.1`), so only the truthfulness of the `# v31` comments needed fixing                     | `.github/workflows/ci.yml:129,177`, `flake-update.yml:22` — comments now `# v31.11.1` (swept into daemon commit `3659c65`)                                                                                                                                                                                    |
| 6   | All 8 lychee broken links repaired: 6 relative links broken by the archival reorganization re-pointed to the targets' real homes, 1 empty-URL link (bare `brandid.BrandName[eventBrand]()` parsed as a markdown link) backtick-quoted                                      | Files: `docs/planning/archived/2026-08-29_20-10_*.md:3`, `docs/status/archived/2026-08-07_00-44_*.md:92`, `docs/status/archived/README.md:6-7`, `docs/status/archived/2026-08-16_11-58_*.md:4`, `docs/status/archived/2026-08-29_18-25_*.md:3`, `docs/status/archived/2026-07-27_10-26_*.md:123`, `docs/status/2026-09-29_05-49_*.md:4`; offline lychee re-run: errors 0 (was 8) |
| 7   | `.buildflow.yml` config regression fixed: my appended `skip_steps:` created a duplicate YAML key whose resolution un-skipped go-structure-linter (6 error findings resurfaced on the next run)                                                                                 | `bb3226a` merges into a single `skip_steps` list; final buildflow run shows `go-structure-linter (skipped via skip_steps config)` and no findings gate failure                                                                                                                                               |
| 8   | AGENTS.md gotcha corrected twice: first recorded the (wrong) concurrent-session theory, then replaced it with the proven root cause and the corollary rules (keep `Name()` a literal; literal can move back behind a constant only if the linter learns to resolve constants)   | `AGENTS.md:126` (final text in `1b571a8`)                                                                                                                                                                                                                                                                    |
| 9   | Gates re-verified after the final tree shape, not assumed: full `scripts/verify.sh` run and a full `buildflow --fix --build-mode=full` run both executed against the finished state                                                                                           | verify.sh log tail: `ALL CHECKS PASSED`; `/tmp/bf-final4.log`: `EXIT=0` (was exit 69 with 4 failed steps at session start)                                                                                                                                                                                    |
| 10  | Coverage re-measured per docs/status conventions (all three modules, root measured as package `.`, never `./...`)                                                                                                                                                             | `go tool cover -func` on fresh profiles: root 99.3%, ssetest 100.0%, sseparse 99.2% (line above)                                                                                                                                                                                                              |

## b) PARTIALLY DONE

1. **Buildflow environment health (the "9 tools unavailable" class).** Triaged the visible
   preflight warnings — stale BuildFlow binary (built at `13d32f7`, repo HEAD `2851a16`),
   `go-licenses` and `interrogate` not in PATH, `prettier`/`dprint` running via
   `nix run nixpkgs#…` without project config. **Not done:** none of the fixes applied —
   rebuilding/reinstalling BuildFlow touches the user's global profile and a shared checkout that
   twelve concurrent sessions may be using; the devShell additions (prettier/dprint) are a policy
   choice about which formatter owns md/json. All fixes are known one-liners, deliberately left.
2. **Flip-war regression safety.** Root cause fixed and documented, but there is **no automated
   guard**: the temporary inotify watcher and the 5-second auto-removal loop I ran during the
   session were session-scoped and are gone. If a future BuildFlow binary changes the linter's
   behavior (or a re-run of an old binary happens), nothing re-alarms except the AGENTS.md note.
   A `brandid-lint` step in CI (or in `scripts/verify.sh`) would close this permanently — not done.
3. **Findings-triage bookkeeping.** The three declined advisory classes (samber/lo suggestions ×3
   tool-modules, encoding/json v2 notes ×4 files, nix-checker's vendorHash-extraction suggestion ×2)
   are declined in this report with rationale but **not encoded where the tools can see it** — they
   will resurface on every future buildflow run as visible noise unless suppressed via
   `.buildflow.yml`-style policy notes or tool-native suppression (if supported). Deferred pending
   a user decision (question g3).
4. **Working-tree hygiene during the flip-war.** Final state is clean and every named fix is in a
   message-bearing commit, but several intermediate states (doc-link fixes, one nix-fmt sweep, one
   flake-meta state) were swept into daemon heuristic commits (`3659c65`, `bdf0bd6`, `10d04d7`,
   `d2086e1`, `5b14057`, `4e50d31`) before I could commit surgically — including one sweep
   (`10d04d7`) that committed the **duplicate method and tab-formatted scripts**, i.e. non-canonical
   content now permanent in history. Per convention, published history stays unrewritten; this
   report is the record. Not done: nothing — this is the honest cost accounting.

## c) NOT STARTED

1. **CHANGELOG `[Unreleased]` entries** for the user-visible fixes (churn root-causes, flake meta,
   doc links) — per docs/status conventions this follows the report; not started.
2. **Harvest of section f) into `TODO_LIST.md` / `ROADMAP.md`** — not started (the convention says
   this follows the report; the list below is the input).
3. **CI confirmation on the pushed tip** — resolved while writing this report: run 37126765073
   finished **green** (see preamble).
4. **BuildFlow binary rebuild + reinstall** — advisory fix (`nix build . && nix run .#reinstall`
   in `/home/lars/projects/BuildFlow`); not started (see b1 and question g2).
5. **`nix run .#govulncheck` sanity on the final tree** — CI has its own vulncheck job, but the
   convention of a local pre-push run wasn't executed this session (low risk: only internal Go
   files changed).

## d) TOTALLY FUCKED UP

1. **I shipped a wrong root cause into AGENTS.md.** After three re-injections I concluded "a
   concurrent Crush session holds a stale event.go buffer" and **wrote that into AGENTS.md** as
   the cause. It was wrong — the injector was buildflow's brandid-lint repair, running inside the
   very gate I was trying to green. I corrected the note an hour later (`1b571a8`), but the wrong
   version was committed and swept upstream in the interim. The tell was visible early: the write
   landed *during* buildflow runs (inotify MODIFY at 15:22:32, mid-gate), which fits a repair step
   and not an idle editor buffer. **Lesson: instrument before theorizing** — the one-liner
   `inotifywait -m event.go` would have pointed at buildflow within minutes.
2. **Whack-a-mole instead of root-cause.** I removed the duplicate and committed **four times**
   (`75476e6`, `6a62140`, `346fd2c`, plus guard-assisted) before reading the linter's source —
   which took two greps once I actually did it (`parseNameReturnValue`, `brands.go`). Each removal
   cycle was a full verify+commit round burned while the injector was still armed. After the
   *second* identical injection the correct move was source-reading, not a third removal.
3. **Duplicate YAML key in `.buildflow.yml`.** Appending a second `skip_steps:` mapping to "add"
   shfmt silently un-skipped go-structure-linter (parser kept the other key), and the next full
   run failed the findings gate with 6 error findings. Caught by the gate itself, fixed in
   `bb3226a`, but it cost a full 5-minute gate cycle and briefly produced findings that should
   never have run (M8 policy).
4. **`mainProgram = null` in flake meta.** I chose the "truthful" null for check packages without
   checking that nix forbids null there (`derivationStrict` rejects a null `NIX_MAIN_PROGRAM` env
   attribute). verify.sh's flake check caught it; one full gate cycle wasted on an eval error I
   could have caught with a 5-second `nix eval .#checks.x86_64-linux.build.meta.mainProgram`
   *before* the gate.
5. **Two buildflow re-runs raced the still-armed injector.** I re-ran the 5-minute gate while the
   duplicate re-injection was still happening on a ~2–20-minute cadence; runs at ~15:22 and
   ~15:25 died on compile errors that my own commits had already fixed seconds before the run
   started flipping the file again. I should have (and eventually did) run a cheap `go build`
   probe immediately before launching the gate, and kept the 5-second guard loop alive until the
   root cause was fixed rather than as a temporary crutch.

## e) WHAT WE SHOULD IMPROVE

| IMP  | Improvement                                                                                                                                                                                        | Priority |
| ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| IMP1 | **Instrument before theorizing.** When a file keeps flipping, first command is `inotifywait -m` (+ fd-scans), not a theory about concurrent sessions. Theories written into AGENTS.md must be labeled "unverified hypothesis" until proven | High     |
| IMP2 | **When a finding makes no sense, read the linter's source immediately.** `brandid-lint`'s BD001 "has no Name() method" on a type that visibly has one was decodable in two greps; that should have been step 1 after re-injection #2, not after #4 | High     |
| IMP3 | **Treat repeated auto-"repairs" as the tool re-creating the problem.** The AGENTS.md hint "tooling dispositions may be fighting" existed; connect it to `--fix` loops early — check what the repair step inserts before removing its output again | High     |
| IMP4 | **Validate gate-config edits against the gate itself, immediately.** After editing `.buildflow.yml`, run `buildflow -s <previously-skipped-step>` (or a dry pipeline compile) to prove the skip list still resolves to one key | Medium   |
| IMP5 | **Fast flake eval before the full gate when meta changes.** `nix eval .#checks.x86_64-linux.build.meta.mainProgram` catches NIX_MAIN_PROGRAM-class errors in seconds; verify.sh's flake check takes minutes | Medium   |
| IMP6 | **Cheap compile probe immediately before any long gate run** in this repo (`go build ./...` — 1s), since a concurrent writer can poison a 5-minute run's early steps                               | Medium   |
| IMP7 | **Encode declined findings where tooling sees them.** Advisory classes declined in prose resurface every run; a policy note in `.buildflow.yml` (the established pattern) or tool-native suppression keeps the signal-to-noise of `--fix` runs usable | Medium   |
| IMP8 | **Run buildflow from inside `nix develop`** (go-licenses, interrogate present) or add the missing binaries to the devShell — removes a recurring preflight-warning class that obscured real findings | Low      |
| IMP9 | **Commit surgical fixes within seconds when a known injector is armed** — the daemon sweep will otherwise enshrine the broken intermediate state (it did, twice today)                             | Medium   |

## f) Up to 50 things we should get done next

**P0 — gate integrity and this session's loose ends**

1. Confirm CI run 37126765073 on `1b571a8` goes green (in progress at report time); if the format
   gate reddens on script content, re-canonicalize via `nix fmt` and commit.
2. Rebuild + reinstall the BuildFlow binary (`cd /home/lars/projects/BuildFlow && nix build . &&
   nix run .#reinstall`) after checking for concurrent BuildFlow sessions — clears the
   binary-freshness advisory (see question g2).
3. Run one clean `buildflow --fix` from inside `nix develop` so `go-licenses`/`interrogate` are on
   PATH; record which warnings disappear.
4. Add `prettier`/`dprint` to the project devShell **or** add a minimal project `dprint.json`
   deciding md/json ownership — stops the "running via nix run WITHOUT project deps" class.
5. Encode the declined advisory findings as policy: `.buildflow.yml` rationale notes (established
   pattern) for samber/lo, json-v2, vendorHash-extraction — or tool-native suppression if the
   tools support it (see question g3).
6. Add a permanent guard against the BD001 repair loop: a `brandid-lint` step in
   `scripts/verify.sh` (binary already in the flake's reach via BuildFlow) or a nix check — so a
   future BuildFlow update cannot silently re-arm the injector.
7. CHANGELOG `[Unreleased]`: entries for `30ce48b` (churn root-cause), `626272b` (flake meta),
   `bb3226a` (config fix), doc-link repair (bdf0bd6).
8. Harvest this section into `TODO_LIST.md` (P0/P1 items) and `ROADMAP.md` (P2/vague) — per
   docs/status conventions, before the session ends.
9. Post-push `git log --stat` review of the daemon sweep commits (`10d04d7`, `3659c65`, `bdf0bd6`,
   `d2086e1`, `5b14057`, `4e50d31`) — confirm no other mislabeled content; record contradictions
   here if found.
10. `grep -rn "eventBrandName"` repo-wide (docs, DOMAIN_LANGUAGE.md, README) — the production
    constant is gone; any remaining prose references are stale (test file reference already fixed).

**P1 — near-term, concrete**

11. Upstream the const-ident limitation to the go-branded-id linter (`parseNameReturnValue` could
    resolve package-level constants) — with verify-before-filing gate first; our workaround note
    is the evidence.
12. Upstream or configure the shfmt disposition in BuildFlow: respect project treefmt config for
    shfmt flags (or support formatter-args in `.buildflow.yml`) so the fleet-wide skip isn't the
    only cure.
13. BuildFlow: BD001 repair should check for an *adjacent existing* `Name()` method before
    inserting its stub (idempotence guard) — the five-injections-a-day failure mode.
14. nix-checker's vendorHash-extraction suggestion (`vendorHash.nix` import): adopt-or-wont
    decision, recorded (current inline form is the documented convention in AGENTS.md).
15. lychee policy: add a repo `lychee.toml` (buildflow's invocation honors it) pinning
    `--offline`-equivalent excludes and deciding whether `docs/**/archived/**` is excluded or
    link-fixed-in-place — prevents the class from resurfacing.
16. go-auto-upgrade jsonv2 findings: record an explicit WONT/deferred decision with the
    GOEXPERIMENT-removal rationale so future sessions don't re-litigate.
17. Review `go.mod`/`go.sum` diffs in daemon sweep commits `5b14057`/`4e50d31` — confirm
    go-mod-update's successful run this session didn't bump anything unintentionally.
18. Grep the repo for remaining empty-URL markdown patterns (`]()` inside prose) — one was found
    by lychee; check for siblings in non-archived docs.
19. Verify `docs/DOMAIN_LANGUAGE.md` and `docs/conformance.md` make no claims invalidated by the
    `eventBrandName` const removal (internal symbol; likely clean — verify, don't assume).
20. Local `nix run .#govulncheck` on the final tree (CI runs it, but a local probe before the next
    push keeps the pre-push gate honest).
21. Decide whether `platforms = lib.platforms.all` should narrow to the flake's declared systems
    (x86_64-linux, aarch64-linux, aarch64-darwin) — honesty vs. churn; record decision.
22. Document the flake-meta convention (homepage/mainProgram=mainProgram/platforms on every
    package; literal-return rule for brand `Name()`) in AGENTS.md Conventions so future packages
    inherit it.
23. After next weekly flake update: confirm statix stays clean on the `checks` attrset and the
    flake-update workflow's treefmt doesn't fight the new grouping.
24. Investigate what the `interrogate` tool is and whether this repo wants it (it's been
    "not installed" in every preflight this session).
25. Confirm go-datastar's weekly compat workflow stays green (ssetest untouched this session, but
    the weekly job re-verifies the consumer chain).

**P2 — hygiene, upstreams, ideas**

26. Move the flip-war diagnostic playbook (inotifywait one-liner + fd-scan) into AGENTS.md gotchas
    as a reusable recipe for future sessions.
27. Consider a crushrc hook that runs `go vet ./...` automatically after any daemon sweep touching
    `.go` files (the AGENTS.md manual rule, automated).
28. ssetest is at 100.0% coverage — consider raising its coverage-gate threshold from 95 so a
    regression can't hide under the old slack.
29. example/datastar `main_test.go:584` manual Filter loop: optional local helper refactor for
    readability without the samber/lo dependency (declined dependency, not declined clarity).
30. ssetest `e2e_test.go:21` same treatment (one small named predicate helper).
31. Revisit sseparse corpus-generator json-v2 migration only when `encoding/json/v2` becomes the
    default-gated stdlib path (ROADMAP; corpus byte-canonicality gate must survive).
32. Add `nix eval .#checks.x86_64-linux.build.meta.mainProgram` (or a loop over all three) to
    verify.sh as a 1-second pre-flake sanity (formalizes IMP5).
33. Consider annotating (not rewriting) the 2026-10-03_14-40 report if it references the old
    duplicate-method theory — docs-health ANNOTATE pass when convenient.
34. BuildFlow idea: findings-gate summary line should distinguish "findings from a step that is
    in skip_steps of a *different* key" — the duplicate-YAML-key failure was invisible.
35. BuildFlow idea: preflight could warn when two formatter dispositions claim the same file
    extension (shfmt vs treefmt-shfmt) — the root cause of today's script churn.
36. Review whether the auto-commit daemon should refuse to sweep `event.go`-class files when
    `go build ./...` fails (it committed non-compiling code twice today) — BuildFlow/crush-config
    proposal.
37. Time-box a background fuzz run (`FuzzReadEvents`, `FuzzSplitSSELines`, 30s each) post-changes —
    parser untouched this session, so purely routine.
38. Check whether `example/README.md` (linked from repaired docs) still matches the two-example
    scope claims after recent sessions.
39. Sweep `TODO_LIST.md` for items this session completed silently (e.g. anything referencing the
    duplicate-method incident) — mark done at hashes.
40. Consider renaming the report-time convention conflict: docs/status/AGENTS.md says "session
    start time", the skill's process says run `date` and use it — align the two texts.
41. Add the three declined advisory classes to a "known noise" section in `.buildflow.yml` header
    comment so `--verbose` triage starts from the policy, not from zero.
42. Verify the `# v31.11.1` comment update survives the next dependabot/renovate-style action bump
    without reverting to bare `# v31` (pin-comment drift detector exists in buildflow — confirmed
    working today).
43. Evaluate `nix run .#coverage-gate` locally once (the convention's canonical command) instead of
    raw `go tool cover` next time — parity with the CI gate's 90/95/95 thresholds.
44. Consider narrowing vulnix noise: the 28 findings are all in nixpkgs-provided build tools — a
    vulnix allowlist/exclude for build-time-only tools, or accept-and-document as fleet policy.
45. Re-run `buildflow doctor` after the BuildFlow rebuild (item 2) to confirm the "9 tools
    unavailable" set shrinks to the genuinely-absent ones.
46. Document in AGENTS.md that `checks.build*` packages are never `nix run` targets (their
    mainProgram is convention, not intent) — prevents a future "why doesn't nix run work" rabbit hole.
47. Verify the final `flake.lock` state from this session's `nix-flake-update` repair steps didn't
    drift vs. the last committed weekly update (diff `flake.lock` against origin if CI green).
48. Add a tiny `scripts/` sanity probe used before long gates: `go build ./...` + `nix flake check`
    dry eval — formalizes IMP6 for all future sessions, not just mine.
49. Schedule the next docs-health pass to reconcile FEATURES.md with this session's fixes (none
    were user-facing features, but the flake meta change is inventory-relevant).
50. Post-incident review item for the fleet: the same brandid-lint BD001 stub-insertion could hit
    every LarsArtmann repo whose brand `Name()` returns a constant — grep sibling repos
    (go-branded-id consumers) for `func (\w+) Name\(\) string { return ` + non-literal and fix
    them before their next buildflow run.

## g) Questions I CANNOT figure out myself

1. **Push/review policy for the churn-fix series:** ~10 commits are already on `origin/master`
   (pushed at ~15:36, mid-session — not by me). Do you want this kind of multi-commit root-cause
   series pre-reviewed before pushing in the future, or is green-gates-then-push acceptable for
   repair work like today's?
2. **BuildFlow binary rebuild:** the running binary is two repo-days stale (`13d32f7` vs
   `2851a16`). I can rebuild and reinstall (`nix build . && nix run .#reinstall`), but that
   checkout is shared with other live sessions and the rebuild would ship whatever is dirty there
   right now. Is another session mid-work in BuildFlow, and do you want the reinstall now or
   after it settles?
3. **Advisory-findings noise policy:** the samber/lo suggestions, encoding/json v2 notes, and
   vendorHash-extraction hint reappear on every buildflow run. Should I formally suppress them
   (`.buildflow.yml` policy notes / tool-native suppression where it exists), or keep them
   visible as deliberate noise so they aren't forgotten?
