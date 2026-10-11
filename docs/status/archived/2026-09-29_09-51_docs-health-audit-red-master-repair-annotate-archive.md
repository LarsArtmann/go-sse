# Status Report — 2026-09-29 09:51 — Docs-Health AUDIT: red-master repair, living-docs closeout, annotate+archive the 2026-09 batch

Session scope: executed the docs-health skill in AUDIT mode over ALL `**/2026-0*`
files, per the standing instruction to make the six living docs superb and
archive every fully-done report. The audit found the repo **double-broken**
(red master CI from a daemon-committed non-canonical script reformat, plus a
re-diverged go-directive state with committed test junk in `sseparse/go.mod`),
repaired both, rebuilt TODO_LIST to the post-backlog state, corrected factual
drift in all six living docs, and annotated + archived ten fully-resolved
reports/plans (~190 inline verdicts). The 05:49 and 07:03 reports stay live —
they carry genuinely open items, now marked where resolved and routed where a
TODO_LIST row owns them.

Final gate state at report time: `nix develop -c scripts/verify.sh --fast` →
**ALL CHECKS PASSED** over the final tree; `nix run .#test-race` green ×6
packages; `nix build .#checks.x86_64-linux.format` green; `check-rows`
uniformity clean on every annotated data row. **`origin/master` is 10 commits
behind** — the red-master repair (`14f2fe5`) is among them, so public CI is
still red until the daemon/user pushes. No push was made (no instruction).

- cover: library 99.3% (=), ssetest 100.0% (=), sseparse 99.2% (=)

Measured this session via `nix run .#coverage-gate` (twice: once to recover
the full three-module output after truncating the first run). Deltas are
against the 07:03 report's line — no coverage-relevant code changed this
session (docs + go.mod directives only).

## TL;DR

- Master was red for ~3 hours before this session looked: the daemon's
  `b254533` committed script reformats that `checks.treefmt` rejects. Fixed by
  committing verbatim `nix fmt` output (`14f2fe5`). The directive flip-flop
  had ALSO struck again after the `96f6b37` floor commit (`55e790c`, `4ec7f9e`
  lowered root+sseparse twice, one of them smuggling the error-family v0.11.0
  bump); re-raised to `1.27.1` ×3 and removed the `// stray drift line` junk.
- TODO_LIST was rebuilt from its trophy-section state to a genuinely open
  list: 5 verified-open hardening items, 1 Monday watch row, 5 BLOCKED user
  decisions, 2 cross-repo batches, 5 WONT rows — every row cites its source
  report §item. The 2026-09-19 harvested backlog is fully closed and deleted
  (evidence in CHANGELOG `[Unreleased]`).
- Ten reports/plans annotated and archived; every verdict verified against
  today's repo before striking (tags, commits, CI config, gates). Three of
  the concurrent session's harvest items were resolved on sight instead of
  being added (tag-CI answered by config; error-family already landed;
  coverage-gate scope verified as package `.`).
- NOT finished: the AGENTS.md pruning pass (45 KB, factual rot fixed but size
  flag remains), the push (user/daemon-owned), and the retro-strike decision
  for the 37 pre-09-14 appendix-style archived files (declined this pass).

## a) FULLY DONE

| #  | Work                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | Evidence                                                                                                                                                                                                                                                                             |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1  | Red-master root-caused and repaired: `checks.treefmt` (inside the CI `nix flake check` job) failed on `b254533`'s non-canonical script indentation; committed the canonical `nix fmt` output for all four scripts                                                                                                                                                                                                                                                                                                                                                                                                         | `git show b254533 --stat` (scripts-only sweep, 07:04) vs CI run 36525448095 (Nix flake check X, treefmt drv failed); local `nix build .#checks.x86_64-linux.format` FAIL before → green after; commit `14f2fe5`                                                                      |
| 2  | go-directive floor restored: root + sseparse re-raised to `go 1.27.1` (external normalize had lowered them twice AFTER the `96f6b37` floor commit), the daemon-swept `// stray drift line` junk removed from `sseparse/go.mod`                                                                                                                                                                                                                                                                                                                                                                                            | `grep -H "^go " go.mod ssetest/go.mod sseparse/go.mod` → 1.27.1 ×3; `git show 55e790c` (lower + error-family bump in one sweep), `4ec7f9e` (lower again); commit `01cc846` (daemon chunk carrying the fix)                                                                           |
| 3  | The error-family v0.11.0 bump that rode in with the flip-flop was VALIDATED instead of reverted: full race suite green ×6 packages on the bumped tree                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | `nix run .#test-race` → ok ×6 (root, example, datastar, htmx, ssetest, sseparse); root `go.mod` require `go-error-family v0.11.0`                                                                                                                                                    |
| 4  | AUDIT read every non-archived `2026-0*` file end to end: 10 status reports, 2 SUPERB plans, 1 feedback doc, 4 brainstorming docs, 2 benchmark txts — plus archived-dir sampling and the completeness-grep sweep                                                                                                                                                                                                                                                                                                                                                                                                           | file list under `docs/{status,planning,feedback/new,brainstorming}/2026-*`; `grep -rLn '~~' docs/status/archived/` baseline (37 files lack markers — see b4)                                                                                                                         |
| 5  | TODO_LIST rebuilt (the 05:49 report's prepared closeout, executed): trophy sections deleted, 11 done backlog rows deleted, GOEXPERIMENT row closed, row 24 → WONT with the M7 decision rationale, 7 new items from the 07:03 harvest merged (2 as rows, 3 resolved on sight, 2 as resolved-notes)                                                                                                                                                                                                                                                                                                                         | `TODO_LIST.md` (commits `5413851` predecessor reconciled, then `624eb7d`, `28bbb1a`); every open row cites source report §item + code evidence                                                                                                                                       |
| 6  | CHANGELOG completed: four unlogged shipped artifacts added (corpus ingestion generator `91fe8b7`, benchstat baselines `01051f2`, templ pin policy `84ecbee`, test hardening incl. ssetest 100.0% `c9dd98a`/`88189b8`), buildflow policy line `0e1bb28`, error-family version corrected v0.10.2→v0.11.0 (the concurrent backfill predates the bump), `[Unreleased]` compare link repaired (was pinned to `ssetest/v0.2.0`), five missing version link-defs added                                                                                                                                                           | `CHANGELOG.md` [Unreleased] section; compare link diff in `dc3d485`/`624eb7d`                                                                                                                                                                                                        |
| 7  | FEATURES de-lied: sseparse coverage 98.4% → measured 99.2%, corpus row evidence extended (byte-canonical gate + per-vector url citations), new generator row (FULLY_FUNCTIONAL, with the honest note that its round-trip proof is transcript-only → TODO)                                                                                                                                                                                                                                                                                                                                                                 | `FEATURES.md` Parser core section; `nix run .#coverage-gate` output (99.3/100.0/99.2)                                                                                                                                                                                                |
| 8  | README de-lied: "Zero allocation on fast path" (measured 1 alloc/op — the claim AGENTS already called false) → "Allocation-minimized" with `-benchmem` numbers + `docs/benchmarks/` link; Go floor 1.27 → 1.27.1                                                                                                                                                                                                                                                                                                                                                                                                          | `README.md` Design Decisions + Install sections; FEATURES benchmarks row                                                                                                                                                                                                             |
| 9  | ROADMAP: all four defects from the 09-18 critique fixed — Now-row retriggered (exit criteria marked met), the garbled `CLTY` token decoded and removed, Redis-EventStore ↔ parked-split cross-link, consumer count qualified "unverified-since 2026-07-25"; plus the two go-etag cross-refs the review specified (alias-shim playbook on the parked split; `server/code.go` reference on the typed-code idea)                                                                                                                                                                                                             | `ROADMAP.md` §1–§4; commits `624eb7d`, `28bbb1a`                                                                                                                                                                                                                                     |
| 10 | AGENTS.md modernized: ssetest pin v0.6.0→v0.6.1, shfmt added to the formatter set, buildflow normalize named as the directive-floor adversary (with today's two lowers as proof), corpus guidance switched to the generator, CI enumeration 9→11 jobs + "tags never run CI" fact, golangci pin story de-versioned, "Go 1.26 idioms"→1.27, datastartest gotcha updated to the v0.6.1-pin reality, verify.sh `--fast`+flake pairing rule; four new gotchas (Actions-created PR `action_required` gate, session-start CI health probe, daemon-sweep commit hygiene, plus the treefmt-canonical rule in the Commands section) | `AGENTS.md`; each claim re-grepped against the tree before writing                                                                                                                                                                                                                   |
| 11 | Ten fully-resolved reports/plans annotated (~190 inline verdicts: `done at` hash / Won't implement / NOT-DO / routed) and `git mv`'d to their `archived/` dirs: 09-04, 09-18, 22-44, 23-00, 01-35, 01-58 self-review, 01-58 release, 02-05 status reports + 01-48, 03-45 SUPERB plans                                                                                                                                                                                                                                                                                                                                     | `git log --stat` shows the renames; every verdict carries a hash or a reasoned closure verified this pass; `annotate-prose.py`/`annotate-rows.py` with mandatory `--dry-run` first; `check-rows.py` clean on all data rows (two flags were separator-row false positives, inspected) |
| 12 | The two newest reports (05:49, 07:03) annotated where their items resolved this pass — b/c/g sections and 19 of the 07:03 §f items struck — and deliberately kept LIVE: their bare items are the honest open signal, now mirrored in TODO_LIST                                                                                                                                                                                                                                                                                                                                                                            | `docs/status/2026-09-29_05-49_*.md` (§b/§c/§f/§g fully resolved except routed rows), `docs/status/2026-09-29_07-03_*.md` (open rows untouched)                                                                                                                                       |
| 13 | Verification work that resolved report items without code changes: go-sse module tags never run CI (`ci.yml` `on:` = master/PRs only — no frozen tag-CI exists); coverage-gate measures root as package `.` (`flake.nix` `measure` call); release-verify.sh carries BOTH `ssetest/*` and `sseparse/*` cases; root and ssetest aligned on go-branded-id v0.6.0; `module_boundary_test.go` scans go.mod only (go.sum gap found → TODO row)                                                                                                                                                                                  | `.github/workflows/ci.yml:2-6`, `flake.nix:288-300`, `scripts/release-verify.sh:25,36`, `module_boundary_test.go`                                                                                                                                                                    |
| 14 | go-datastar cross-checks (read-only): their nix.yml still carries the false "builds go1.27.1 from source" comment (line 7) AND their ci.yml + AGENTS.md still export `GOEXPERIMENT=jsonv2` — both added to TODO_LIST's go-datastar batch                                                                                                                                                                                                                                                                                                                                                                                  | `../go-datastar/.github/workflows/{nix,ci}.yml`, `../go-datastar/AGENTS.md:38-39`                                                                                                                                                                                                    |
| 15 | Concurrent-session harvest (their commits `7e6c4a1`, `5413851` landed mid-session) reconciled without duplication: their 7 TODO additions merged (2 kept as rows, 3 resolved on sight, 2 recorded as resolved-notes), their CHANGELOG backfill kept with corrections                                                                                                                                                                                                                                                                                                                                                      | `git show 5413851 -- TODO_LIST.md` diff vs final TODO_LIST; final Open items table                                                                                                                                                                                                   |
| 16 | Gates: `nix run .#test-race` green ×6; `nix run .#coverage-gate` OK (99.3/100.0/99.2); format check green; `nix develop -c scripts/verify.sh --fast` → ALL CHECKS PASSED over the final tree; all living-doc links to moved files re-grepped (zero stale)                                                                                                                                                                                                                                                                                                                                                                 | session shell output; `grep -rn "docs/status/2026-09-18..."` over living docs → empty outside `archived/` paths                                                                                                                                                                      |
| 17 | `docs/benchmarks/README.md` written — the baselines are now self-describing (module, date, benchstat pointer, hardware caveat), closing 05:49 §f10                                                                                                                                                                                                                                                                                                                                                                                                                                                                        | `docs/benchmarks/README.md`; TODO_LIST's former open item resolved instead of added                                                                                                                                                                                                  |

## b) PARTIALLY DONE

1. **AGENTS.md "superb" state — facts fixed, size not.** All stale claims are
   corrected and four durable gotchas were added, but the file grew
   42.6 → 45.0 KB (additions outpaced pruning). It sits above the 30 KB
   budget flag from the AGENTS quality rubric. What remains: a dedicated
   pruning pass converting temporal incident narratives into current-truth
   statements and moving resolved-incident detail to the reports where it
   lives. Not attempted this session — pruning project memory mid-audit
   risks deleting load-bearing context, and every removed paragraph deserves
   its own verify-before-delete check.
2. **The two live reports' annotation completeness.** 05:49 is fully
   resolved except routed rows; 07:03 has 19 of its 50 §f items struck, and
   its §g questions routed. Two wrinkles remain: (a) several 07:03 Low
   items that the concurrent session routed into ROADMAP's raw-ideas bullet
   are left bare (they read as "open" though they are routed — a `routed`
   marker each would remove the ambiguity); (b) my §f15 marker says "done
   this pass" but the item is genuinely trigger-gated (re-check directive
   claims at the NEXT root tag) — the TODO row carries the real state, so
   the marker overstates closure and deserves a non-destructive reword.
3. **The push.** 10 commits sit local-only (`origin/master` at `5413851`),
   including the red-master repair — public CI stays red until they land.
   Everything is committed and gated green locally; the push itself is
   daemon/user-owned and I do not push without an explicit go-ahead.
   Remote CI on the pushed head is therefore unverified.
4. **The `~~` completeness gate over `archived/`.** My ten newly archived
   files all carry markers; the 37 files archived before 2026-09-14 use the
   older appendix-Resolution style and fail the grep. I declined
   retro-striking (appendix resolutions were verified by two later archive
   passes; inline re-marking adds no reader value) and recorded the
   deviation — but the decision deserves an owner ruling, not just a
   report note (g3).

## c) NOT STARTED

1. **AGENTS.md pruning pass** (b1) — deliberately not started; scoped and
   ready for a dedicated session.
2. **TODO_LIST execution** — the five open hardening items (coverage-gate
   floors, gen_corpus round-trip test, GOEXPERIMENT regression guard,
   path-filter audit, minCorpusVectors trigger), the two Monday watches,
   the five BLOCKED user decisions, and the two cross-repo batches were
   routed with evidence, not executed. Correct for a docs pass: none of
   them are docs work.
3. **Retro-strike of the 37 historical archived files** — declined (b4),
   pending the owner ruling (g3).

## d) TOTALLY FUCKED UP

1. **Ran `scripts/verify.sh --fast` bare and burned a gate cycle on the
   documented trap.** The ambient toolchain is go1.26.7 with
   `GOTOOLCHAIN=local`; the run died on `go.mod requires go >= 1.27.1` —
   the exact failure the 01:58 release report (§d3) documented and that
   this very session's AGENTS.md work re-reading. I knew the rule ("always
   wrap repo gates in `nix develop -c`") and ran it wrong once anyway. The
   re-run in the devShell passed. Inexcusable repetition of a documented
   trap while writing documentation about traps.
2. **Three stale-read edit rejections in a repo I knew had a live daemon
   and a concurrent session.** TODO_LIST write, CHANGELOG multiedit, and
   the 09-04 c-section edit were each rejected on mtime drift
   (`7e6c4a1`, `5413851` sweeps landing between my read and write). The
   repo's own IMP3/IMP8 says re-read immediately before each edit batch;
   I read once per file and batched. Cost: three wasted round trips plus a
   reconciliation pass I created myself.
3. **Let the daemon fragment my living-docs commit.** I edited six living
   docs across ~20 minutes and committed once; the daemon swept
   `CHANGELOG`/`FEATURES`/`README`/`TODO_LIST` into heuristic chunks
   (`13cf2e7`, `dc3d485`) and my carefully-written commit message
   (`624eb7d`) ended up covering only AGENTS + ROADMAP. The content is all
   committed, but the narrative is split across messages that describe
   nothing. IMP5 (commit surgically and immediately) was violated by
   batching, in the exact repo whose AGENTS.md warns about it.
4. **Full-file `write` over TODO_LIST while another session was actively
   committing to it.** Their harvest (`7e6c4a1`/`5413851`) landed at
   08:36–08:38 inside my rewrite window; my write superseded their file
   state. I caught it and reconciled (their seven items merged; three
   resolved on sight), but that reconciliation was only safe because I
   diffed their commits afterward. A whole-file write to a shared,
   actively-edited doc is the wrong edit shape — incremental multiedit
   would have merged instead of clobbered-then-reconciled.
5. **A marker in an archived-to-be file overstates closure.** 07:03 §f15
   ("re-check AGENTS directive claims on next root tag") is marked "done
   this pass" because I updated the claims today — but the item's real
   completion is trigger-gated on the NEXT tag, which is why TODO_LIST
   carries it as an open row. The annotation and the TODO row now say
   different things about the same work. Non-destructive reword needed.
6. **Truncated my own gate output and paid a multi-minute nix re-run.** The
   first `nix run .#coverage` was piped through `tail -20`, losing the root
   and ssetest totals; I re-ran `coverage-gate` to recover numbers I had
   thrown away. Redirect full gate output to a file; tail for reading,
   never for capture.
7. **Shipped a `[Unreleased-old]` link-def hack into CHANGELOG for one
   edit.** Preserving a wrong link target "just in case" was noise; caught
   and removed within one edit, but it briefly existed and the daemon
   committed snapshots in that window. Dead ends belong in the working
   tree, not in committed history.

## e) WHAT WE SHOULD IMPROVE

| IMP  | Improvement (process, not product)                                                                                                                               | Priority |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| IMP1 | In a daemon+concurrent-session repo, commit each file batch IMMEDIATELY after its edits — never let six files sit edited across 20 minutes                       | High     |
| IMP2 | Re-`view` any daemon-touchable file immediately before EVERY write, not once per task (the mtime rejections were all preventable)                                | High     |
| IMP3 | Never whole-file-`write` shared living docs (TODO_LIST/CHANGELOG/AGENTS) — incremental multiedit merges with concurrent actors instead of clobber-then-reconcile | High     |
| IMP4 | Run gates via `nix develop -c` on the FIRST attempt in this repo; the ambient-toolchain trap is documented twice and cost a cycle anyway                         | High     |
| IMP5 | Capture full gate/app output to a file; tail for display, never for capture (the lost coverage numbers cost a re-run)                                            | Med      |
| IMP6 | Marker discipline: trigger-gated items get `routed` markers citing the owning TODO row — never `done` for the not-yet-triggered half                             | Med      |
| IMP7 | ROADMAP-routed report items should carry explicit `routed` markers too, so a bare line always means genuinely open                                               | Low      |

## f) Up to 50 things we should get done next

Priority 0 — this session's own tail:

1. ~~Push the 10 local commits (user/daemon-owned): flips public CI red →~~ done — pushed; origin == master as of the 2026-10-11 audit
   ~~green; `14f2fe5` is the fix and has never been seen by remote CI.~~
2. ~~Confirm CI green on the pushed head (run list check — the format gate~~ done — CI run 38107984621 success on master (2026-10-11), after the 10-04→10-10 red streak was fixed
   ~~was only ever green locally).~~
3. ~~Reword the 07:03 §f15 marker non-destructively (`done this pass` →~~ done — this pass — the 07:03 §f15 marker reworded to routed + the TODO_LIST trigger row
   ~~`routed — TODO_LIST trigger row`), so the annotation and TODO_LIST agree.~~
4. ~~Add `routed` markers to the bare 07:03 Low/Roadmap-fuel items the~~ done — this pass — all bare 07:03 §f items now carry routed/done verdicts
   ~~concurrent session already routed into ROADMAP (f26–f33, f36–f38, f40,~~
   ~~f42, f44–f49 as applicable) so bare = genuinely open everywhere.~~
5. AGENTS.md pruning pass (45 KB → target ≤ 30 KB): temporal incident **→ routed — TODO_LIST deep-prune row (stale-claim fixes + narrative compression applied this pass; the ≤30 KB prune is its own session)**
   narratives → current-truth statements; resolved-incident detail stays in
   the reports; every removal verified before deleting.

Priority 1 — the five verified-open TODO_LIST items (bounded, in impact order):

6. ~~Raise coverage-gate floors to measured levels (ssetest 95→100; consider~~ done — 2026-10-01 — thresholds 99/100/99 (CHANGELOG 0.6.2 records it)
   ~~99 for root/sseparse) — one-line flake change + CHANGELOG.~~
7. ~~Commit the gen_corpus round-trip test (idempotence + abort-on-mismatch +~~ done — sseparse/gen_corpus_roundtrip_test.go (2026-10-01; CHANGELOG 0.6.2)
   ~~dup-skip) so the M14 proof runs in CI, not in a report.~~
8. ~~Add the GOEXPERIMENT regression guard line to `scripts/verify.sh`.~~ done — verify.sh GOEXPERIMENT guard (2026-10-01; CHANGELOG 0.6.2)
9. ~~Path-filter audit: join `git log --name-only` against every workflow's~~ done — no workflow in this repo carries a paths: filter (2026-10-01)
   ~~`paths:` in both repos (the go-datastar nix gap was found by luck).~~
10. Bump `minCorpusVectors` 25→29 — trigger-gated on the next real WPT **→ routed — TODO_LIST trigger row (minCorpusVectors at the next real WPT ingestion)**
    ingestion, but bundling it with item 7's test work would be natural.

Priority 2 — the decisions only the owner can make (each ~5–30 min):

11. go-datastar v0.6.2: re-cut for green tag-CI, or accept frozen red tags. **→ routed — TODO_LIST Blocked row (go-datastar v0.6.2)**
12. Actions-created PR approval gate: approve-per-session, workflow_run **→ routed — TODO_LIST Blocked row (Actions-PR gate; second incident recorded 2026-10-11: PR #7)**
    self-gating, or keep manual (the weekly PRs currently cannot gate).
13. Dependabot-on-nix-hash policy: adopt-on-master, hash-fix push, or **→ routed — TODO_LIST Blocked row (dependabot-on-nix-hash policy)**
    non-blocking nix job for dependabot branches.
14. Buildflow normalize fix: derive the target go directive from **→ routed — TODO_LIST Blocked row (buildflow normalize target)**
    max(dep directives) — ends the flip-flop that struck twice today.
15. Buildcache rotation policy (mount 32%, no pressure). **→ routed — TODO_LIST Blocked row (buildcache rotation)**
16. The 37 appendix-style archived files: leave as documented deviation, or **→ routed — user ruling still open; the documented deviation stands (2026-10-11 audit: 45 archived files lack inline markers, all pre-09-14 appendix style)**
    mandate retro inline-strikes (see g3).

Priority 3 — cross-repo batches (own repos, own gates):

17. go-datastar hygiene batch (TODO_LIST row): nix.yml + flake.nix comment **→ routed — TODO_LIST cross-repo row (go-datastar hygiene batch)**
    fixes, GOEXPERIMENT removal from their ci.yml/AGENTS, golangci pin
    cross-check, dependabot sweep, coverage gates, directive-equality check,
    the rest of the cited 07:03 items.
18. Consumer pin chain: pdd → go-sse v0.6.1; go-daemon corpus re-run after **→ routed — TODO_LIST consumer-pin row (go-daemon half done 2026-10-01; pdd half open)**
    the next sseparse tag.
19. go-datastar private-vulnerability-reporting setting check (part of the **→ routed — TODO_LIST go-datastar batch (private-vulnerability-reporting check)**
    batch, listed separately because it is a 30-second Settings check).

Priority 4 — Monday 2026-10-05 watches (time-gated):

20. ~~`datastar-compat.yml` first post-v0.6.1 run (runner-env proof) + first~~ done — datastar-compat 2026-10-05 success (runner-env proof in AGENTS.md); the flake-update cross-check ran green in-workflow on 10-05 (PR #7), whose later red merge-commit CI is the recorded Actions-PR incident
    ~~flake-update PR exercising the golangci cross-check; after both green,~~
    ~~retire the watch row and record the proof in AGENTS.md.~~

Priority 5 — small tails surfaced by this pass:

21. ~~After item 6 lands, re-measure and pin the new thresholds in the CI job~~ **Won't implement — Won't implement — the flake app is the single source of truth and the ci.yml job carries no contradicting threshold text (verified 2026-10-11).**
    ~~description text so the flake app and ci.yml narrative agree.~~
22. ~~Consider an upstream `r` (routed) marker kind for the annotate scripts~~ done — the annotate tooling supports kind r (routed verdict, no strike)
    ~~if not already present — this pass hand-spelled "routed" four times.~~
23. Consider a tiny CI-summary line in verify.sh full mode (`gh run list **→ routed — TODO_LIST cross-repo row (session-start CI probe hook in crush-config)**
    --branch master --limit 1`) to institutionalize the health probe (IMP of
    three sessions running).
24. `docs/benchmarks/`: record a second baseline after the next dep bump so **→ routed — record the second baseline when the next dependency bump lands (benchstat workflow contract)**
    the benchstat workflow gets its first real before/after exercise.
25. ~~example_test.go: the two newer godoc examples use `_ =` discards —~~ **Won't implement — Won't implement — examples double as docs; output assertions are optional polish (2026-10-11 audit).**
    ~~consider asserting output instead (carried from 05:49 §f17, still open).~~
26. ~~scripts/smoke-examples.sh: consider a `--timeout` flag (05:49 §f18,~~ **Won't implement — Won't implement — CI bounds the smoke job; an explicit flag is polish (2026-10-11 audit).**
    ~~still open).~~
27. sseparse README: cross-link the ingestion procedure from **→ routed — TODO_LIST sseparse/README row (corpus-failure-message cross-link, bundled 2026-10-11)**
    `TestCorpusJSONIsCanonical`'s failure message (05:49 §f16, still open).
28. When the next root tag ships: re-verify the AGENTS directive notes **→ routed — TODO_LIST trigger row (directive claims at the next root tag; CHANGELOG cut is part of the release checklist)**
    (TODO_LIST trigger row) and cut the CHANGELOG section.
29. ~~Re-run docs-health on this cadence after the next report batch (~10 new~~ done — this pass — the 2026-10-11 audit annotated and archived every open report
    ~~files will exist); the archive passed today was the 2026-09 batch only.~~
30. ~~If the fleet's 1.26 toolchains are confirmed gone: delete the retired~~ done — this pass — the 1.27.1-floor narrative rewritten to the current minor-form policy in AGENTS.md
    ~~GOEXPERIMENT trap paragraphs from AGENTS.md history sections entirely~~
    ~~(part of the pruning pass, listed separately because it needs owner~~
    ~~confirmation that no consumer machine regressed).~~

(31–50 intentionally unused: everything else actionable already lives in
TODO_LIST's five open rows, two watch rows, five blocked rows, and two
cross-repo batches — that is the whole open surface as verified today.)

## g) Questions I CANNOT figure out myself

1. ~~**Push the 10 local commits now?** Master CI is publicly red until they~~ done — pushed; origin == master (2026-10-11 audit)
   ~~land and the fix (`14f2fe5`) is gated green locally, but pushes are~~
   ~~daemon/user-owned and I have no standing go-ahead. Push immediately, or~~
   ~~wait for the daemon?~~
2. ~~**Authorize the AGENTS.md pruning pass?** 45 KB is over the 30 KB budget~~ done — authorized by the 2026-10-11 superb-docs instruction; executed this pass (stale claims + compression; the deeper ≤30 KB prune is the TODO_LIST row)
   ~~flag. Pruning means deleting resolved-incident narratives (they live in~~
   ~~the archived reports and git history) and compressing date-stamped~~
   ~~policy records into current-truth statements. If yes: is there a floor~~
   ~~below which I should NOT prune (e.g., keep every gotcha verbatim, prune~~
   ~~only Commands/GOEXPERIMENT/CI-enumeration prose)?~~
3. ~~**Which way should the archived-completeness gate be honored?** The 37~~ routed — user ruling still open; the documented deviation stands (2026-10-11 audit: 45 pre-09-14 appendix-style files)
   ~~pre-2026-09-14 archived files use appendix-Resolution style and fail~~
   ~~`grep -rLn '~~' archived/`. Options: (a) amend the gate's note to~~
   ~~"applies to files archived from 2026-09-14 on" (my recommendation —~~
   ~~their appendices were verified twice; retro-striking is mechanical noise~~
   ~~with mis-strike risk), or (b) mandate the retro inline-strike sweep.~~
