# Status Report — 2026-09-29 05:49 — SUPERB plan execution closeout (M3–M17)

Session scope: execute the remaining SUPERB-plan tasks
([plan](../planning/2026-09-29_03-45_SUPERB-consumer-chain-and-hygiene-closeout.md))
after auditing that M1/M2/M9/M10/M12 were already landed by the earlier
04:2x session. This session verified + pushed that prior work, then executed
M3–M17: the pdd cascade finish, GOEXPERIMENT removal, two new CI gates, two
recorded policy decisions, direct unit tests, ssetest to 100%, the corpus
ingestion generator, the smoke app + shfmt, the templ pin policy, and
benchmark baselines.

Gate state: `scripts/verify.sh` full → ALL CHECKS PASSED (after M4);
`nix flake check` → all checks passed (after M15); `verify.sh --fast` → ALL
CHECKS PASSED (after M14, the last Go-code change; M16/M17 were docs-only).
A final full gate over the accumulated tail commits is still pending (§b2).
Note: a concurrent actor left uncommitted `go 1.27.1` hand-raises in
`go.mod` + `sseparse/go.mod` (see §d6/§g1) — the gate has NOT been run
against that state, and per the directive-equality rule it would redden
while ssetest stays at `go 1.27`.

- cover: library 99.3% (=), ssetest 100.0% (+2.0), sseparse 99.2% (=)

Measured this session (fresh `go test -coverprofile` + `go tool cover
-func`, root as package `.`). Cross-repo coverage not re-measured: this
session changed nothing in go-datastar/go-daemon/pdd beyond verifying and
pushing the earlier session's commits.

## TL;DR

- The consumer chain is now verified end-to-end, not just released:
  go-datastar carries v0.6.1/v0.4.0 pins with `datastartest/v0.6.1` tagged
  and replaces dropped; go-daemon runs its framing parser against the
  sseparse corpus; pdd is cascaded, race-green, and pushed.
- `GOEXPERIMENT=jsonv2` is gone from every live surface; the hermetic flake
  checks rebuilt without it and stayed green.
- CI gained the two gates that were local-only (coverage thresholds,
  shellcheck); ssetest reached 100.0% statement coverage; the corpus got a
  dispatch-validating, idempotent WPT-ingestion generator — whose proof run
  caught a real append-aliasing bug before it could ship.
- Two recurring decision-noise items became recorded policy (golangci pin =
  WONT with control chain; go-structure-linter = skip with rationale).
- NOT finished: the TODO_LIST closeout rewrite (raced an external edit, §b1),
  the final full gate + remaining push, and this report's §f harvest.

## a) FULLY DONE

| #   | Work                                                                                                                                                                                      | Evidence                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| a1  | State audit: M1/M2/M9/M10/M12 confirmed already-done, then go-sse's 2 unpushed commits verified + pushed                                                                                  | go-datastar `git tag` has `v0.6.1` + `datastartest/v0.6.1`, replaces absent from both `go.mod`s; go-daemon `36c2ecc` + `sse_conformance_test.go` + sseparse require; go-sse pushed `c97f0f6..6ca06d7`; root+ssetest race suites green before push                                                                                                                                                                                                                                                                                               |
| a2  | M3 finished: pdd cascade verified and pushed                                                                                                                                              | `rg CodeSSEScanFailed` = 0 hits (exit 1); `go build ./... && go test ./... -race` green in pdd devshell; pushed `d2efa06..1606496`                                                                                                                                                                                                                                                                                                                                                                                                              |
| a3  | M4: `GOEXPERIMENT=jsonv2` removed from all 25 live sites (flake devShell + ci shell + 3 hermetic checks + 7 apps, ci.yml ×8, datastar-compat.yml, 3 scripts, local `.envrc`); docs un-lie | `flake.nix`, `.github/workflows/ci.yml`, `datastar-compat.yml`, `scripts/{verify,smoke-examples,release-verify}.sh`, `.envrc`, AGENTS/README/CONTRIBUTING/FEATURES; full gate green with hermetic derivations rebuilt flag-less; CHANGELOG Changed line; squashed 3 daemon commits into `1c0fc85`                                                                                                                                                                                                                                               |
| a4  | M5: CI `Coverage gate` job                                                                                                                                                                | ci.yml `coverage-gate` job runs `nix run .#coverage-gate` (thresholds single-sourced in the flake app); dry-run: 99.3/98.0/99.2 vs 90/95/95 OK; actionlint clean; commit `41aac66`                                                                                                                                                                                                                                                                                                                                                              |
| a5  | M6: CI `Shellcheck` job                                                                                                                                                                   | ci.yml `shellcheck` job (`shellcheck scripts/*.sh`); local run clean before landing; same commit                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| a6  | M7: golangci-lint pin decided — WONT with recorded control                                                                                                                                | flake.lock pins the effective version; `scripts/golangci-pin.sh` verify-check + flake-update cross-check surface skew; rationale in AGENTS.md Gotchas; TODO 24 closes as WONT                                                                                                                                                                                                                                                                                                                                                                   |
| a7  | M8: go-structure-linter decided — skipped as recorded policy                                                                                                                              | New policy-only `.buildflow.yml` (`skip_steps: [go-structure-linter]` + rationale: 9 findings, one class, all flagging the deliberate flat single-package layout); `buildflow --dry-run` shows "skipped via skip_steps config"; AGENTS.md gotcha added; commit `0e1bb28`                                                                                                                                                                                                                                                                        |
| a8  | M11: direct unit tests for `forEachLine` (10-case table) and all three examples' `listenAddr()`                                                                                           | `event_lines_internal_test.go` (CR/LF/CRLF/empty/trailing/sandwich rules); `example/{,datastar/,htmx/}main_test.go` default + PORT override with `t.Setenv` forcing both ways; `verify.sh --fast` green after formatter fix; commit `c9dd98a`                                                                                                                                                                                                                                                                                                   |
| a9  | M13: ssetest coverage 98.0% → **100.0%**                                                                                                                                                  | 5 branch tests in `collect_internal_test.go`: count-guard nil, doRequest bad-method Fatalf, doRequest canceled-ctx Fatalf, CollectWithTimeout hang→Fatalf (real socket, 100ms), CollectN short-body read-error Fatalf (Content-Length lie — clean EOF is deliberately not an error); new `fatalTB` + `requireFatalPanic`; race suite green; FEATURES updated; commit `88189b8`                                                                                                                                                                  |
| a10 | M14: corpus WPT-ingestion generator, wired via `go:generate`, proven idempotent                                                                                                           | `sseparse/gen_corpus.go` (`//go:build ignore`), directive in `corpus.go:20`; six-step proof: no-pending run = zero diff; wrong-events pending ABORTS on dispatch mismatch; correct pending ingests (29→30 vectors, canonical bytes); identical-dup re-run = no-op; corpus gates green post-ingest; tree restored clean. Proof caught the ingest append-aliasing bug (local slice header) before it could ship — fixed by returning the grown slice. Procedure documented in `sseparse/README.md` §"Ingesting new WPT vectors"; commit `91fe8b7` |
| a11 | M15: `nix run .#smoke` app + shfmt in the format gate                                                                                                                                     | flake `smoke` app (go/curl/coreutils runtimeInputs) — "ALL EXAMPLE SMOKE CHECKS PASSED"; `programs.shfmt.enable`; 4 scripts reformatted (tabs→2-space, zero semantic), shellcheck clean after; full `nix flake check` green; CHANGELOG; commit `0bd6e69`                                                                                                                                                                                                                                                                                        |
| a12 | M16: templ CLI pin policy documented                                                                                                                                                      | CONTRIBUTING §"The templ CLI pin": both pin sites (ci.yml `-check` drift alarm + devShell `pkgs.templ`), why `@version` form is mandatory, bump-both-and-regenerate procedure; commit `84ecbee`                                                                                                                                                                                                                                                                                                                                                 |
| a13 | M17: benchmark baselines + benchstat workflow                                                                                                                                             | `docs/benchmarks/2026-09-29-{root,sseparse}.txt` (count=6; 7 root benches + BenchmarkReadEvents); CONTRIBUTING §"Benchmark regression tracking" (pinned `benchstat@v1.0.0`, why count=1 is useless, re-record rule); FEATURES links the baselines; commit `01051f2`                                                                                                                                                                                                                                                                             |
| a14 | README consumer-lie fixed en passant                                                                                                                                                      | README claimed "Go 1.26.7+ with GOEXPERIMENT=jsonv2" — now "Go 1.27+, no build tags" (part of a3)                                                                                                                                                                                                                                                                                                                                                                                                                                               |

## b) PARTIALLY DONE

1. ~~**TODO_LIST closeout rewrite — content prepared, write rejected.** The
   multiedit converting the open/harvested tables to the resolved state was
   rejected because the file changed on disk after my read (mtime jumped
   04:25:48 → 04:25:59 with NO working-tree diff — an external/committed
   change, see §d6). On re-read, rows 20/21/28/29/30 were already gone from
   the harvested table (someone else closed them), so my prepared
   replacement text is stale anyway. What remains: rewrite against the
   current 11-row table (14–18, 22–25, 27, 31), close row 41 (GOEXPERIMENT),
   move row 24 to WONT, resolve the Cross-repo datastartest-replaces WONT
   row (done at `datastartest/v0.6.1`), and update the wait-state row.
   Nothing was written; the file is untouched by me — deliberately, rather
   than forcing a stale rewrite.~~ done — the docs-health pass rebuilt TODO_LIST on 2026-09-29 (5413851, 624eb7d, 28bbb1a): every closed row deleted, row 24 → WONT, open §f items routed
2. ~~**Final full gate + push over the tail commits.** The last full
   `scripts/verify.sh` predates M16/M17 (docs-only) and the daemon's last
   commits; `origin/master..master` currently shows only daemon commit
   `0b52e9c` (the daemon has been pushing; I pushed explicitly through
   `6ca06d7` and pdd). One more full gate + push remains — AND the working
   tree currently carries the foreign go-directive edits (§d6), which the
   directive-equality gate would reject; that state must be resolved first
   (§g1).~~ done — the 07:03 session aligned the directives (96f6b37) and gated green; this pass repaired the later format-gate red (14f2fe5)

## c) NOT STARTED

| Item                                                            | Why                                                                                                   |
| --------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| ~~M18: watch Monday's `datastar-compat.yml` run (next 2026-10-05)~~ routed — TODO_LIST watch row | ~~Wait-state by design; mechanics already hand-verified; the run is the runner-env proof.~~               |
| ~~M19: enable GitHub private vulnerability reporting~~ done — enabled via the REST endpoint | ~~USER-ONLY (Settings UI; REST rejects it). Surfaced, needs you.~~ (the 422 pointed at the wrong endpoint) |
| ~~M20: buildcache rotation policy decision~~ routed — TODO_LIST Blocked row | ~~USER-ONLY (host policy; mount was 32%, no pressure).~~                                                  |
| ~~go-datastar/go-daemon/pdd coverage bullets for the report line~~ NOT-DO — no changes made in those repos this session, so there is nothing to measure | ~~No changes made in those repos this session (verification + push only) — nothing to measure honestly.~~ |

## d) TOTALLY FUCKED UP

1. **Sloppy multiedit old_string shipped a broken imports edit** (M13): 2 of
   3 edits applied, the imports edit failed on a stray character in my
   match text, and I didn't notice until `go test` failed with undefined
   symbols. The tool told me "1 failed" and I moved on — should have
   verified immediately.
2. **Nearly overwrote a 300-line test file** (M11): `write` to
   `example/datastar/main_test.go` without checking it existed (it did, with
   the memStore/fan-out integration tests). The tool's read-before-write
   rejection saved me; the cost was one wasted round trip. Should have
   globbed the example dirs first.
3. **Wrote preemptive `//nolint:paralleltest` directives that were dead on
   arrival** (M11): paralleltest is t.Setenv-aware, so nolintlint flagged
   all four as unused (a gate failure had they survived). Removed them;
   lesson: run the linter before suppressing it.
4. **First generator proof run tested the wrong branch** (M14): my printf
   escaping turned `\n` into literal newlines inside JSON strings, so the
   "wrong events" pending file died on JSON decode, not on the dispatch
   mismatch I meant to prove. The heredoc retry proved the real branches.
5. **Invented a statistic** (M17): I wrote "noise on this hardware is
   roughly ±10%" into CONTRIBUTING with zero measurements behind it.
   Caught it during self-review and deleted the sentence before commit —
   but it nearly shipped as authoritative-sounding lore.
6. **Missed that the repo has a concurrent actor** until late: my TODO_LIST
   rewrite raced an external edit (§b1), my commits are being pushed by
   something that is not me (origin caught up without my push), and the
   working tree now carries `go 1.27` → `go 1.27.1` hand-raises in root +
   sseparse `go.mod` that I did not make (05:4x). None of my work touched
   those files, nothing was reverted, and the state is flagged here instead
   of "fixed" — per the never-revert-what-you-didn't-author rule. But I
   should have noticed the push drift after the first commit instead of
   assuming the push state from the session-start summary.
7. **Stale-read risk taken with TODO_LIST**: I composed the closeout from a
   read taken ~11 seconds before the external change; batching a large
   multiedit over a file other sessions touch is asking for exactly the
   rejection that happened.

## e) WHAT WE SHOULD IMPROVE

| IMP | Improvement                                                                                                                                                         | Priority |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| 1   | Glob for existing test files before any new test-file write; append-anchor instead of whole-file writes                                                             | High     |
| 2   | Run the linter on new test code BEFORE adding nolint directives (nolintlint then guards, not churns)                                                                | High     |
| 3   | Re-read externally-mutated files (TODO_LIST, go.mod, CHANGELOG) immediately before each edit batch, not once per task                                               | High     |
| 4   | Ship generator proofs as committed tests, not session transcripts: an idempotence/abort/ingest round-trip test for gen_corpus would make the proof repeatable in CI | Medium   |
| 5   | Treat "tool reported partial failure" as a stop-and-verify signal, never a footnote                                                                                 | High     |
| 6   | Push per milestone rather than accumulating 10+ local commits and relying on the daemon's push behavior                                                             | Medium   |
| 7   | When measuring claims for docs (noise figures, budgets), either measure or omit — never estimate into a guide                                                       | High     |
| 8   | Cross-session protocol: check `git log origin/master..master` + `git status` at session start AND before each task                                                  | Medium   |

## f) Up to 50 things we should get done next

Priority 1 — finish this session's own tail:

1. ~~Resolve the foreign `go 1.27.1` go.mod edits (accept + align ssetest, or~~ done — accepted + aligned all three at 1.27.1 (84c3aae, 96f6b37); re-raised again after later normalize lowers (01cc846)
   ~~revert — owner's call, §g1), then run the directive-equality gate.~~
2. ~~Redo the TODO_LIST closeout against the current 11-row harvested table~~ done — this pass rebuilt TODO_LIST (5413851, 624eb7d, 28bbb1a)
   ~~(content in §b1 is 90% ready; re-anchor rows 20/21/28/29/30 as~~
   ~~already-removed).~~
3. ~~Final full `scripts/verify.sh` + `nix flake check` over the accumulated~~ done — the 07:03 session gated and pushed the tail; this pass repaired the format-gate red it found (14f2fe5)
   ~~commits; push the remainder explicitly.~~
4. ~~Harvest this report's §f into TODO_LIST/ROADMAP per the conventions.~~ done — 5413851 harvested the 07-03 items; this pass completed the harvest and annotated/archived the reports

Priority 2 — harden what this session shipped:

5. ~~Commit a gen_corpus round-trip test (idempotence + abort-on-mismatch +~~ done — routed — TODO_LIST open item (gen_corpus round-trip test)
   ~~dup-skip) so the M14 proof runs in CI instead of living in this report.~~
6. ~~Add a cheap `rg GOEXPERIMENT scripts/ flake.nix` regression guard to~~ done — routed — TODO_LIST open item (GOEXPERIMENT regression guard)
   ~~verify.sh so the removed flag cannot creep back silently.~~
7. ~~Watch the first post-session flake-update PR: it is the first live~~ done — routed — TODO_LIST watch row (first post-cross-check flake-update PR)
   ~~exercise of the golangci cross-check with the WONT decision recorded.~~
8. ~~Bump `minCorpusVectors` (25) when the next real WPT ingestion lands, so~~ done — routed — TODO_LIST open item (minCorpusVectors trigger)
   ~~the integrity floor tracks the corpus's actual size.~~
9. ~~ssetest is at 100% — add a note to the coverage-gate thresholds discussion~~ done — routed — TODO_LIST open item (coverage-gate floors)
   ~~whether ssetest should be pinned at 100 (currently 95) so a regression~~
   ~~reddens immediately.~~
10. ~~Consider a `docs/benchmarks/README.md` one-liner (what the files are,~~ done — this pass — docs/benchmarks/README.md written
    ~~which module, hardware) so the baselines are self-describing.~~
11. ~~datastar-compat: after a green 2026-10-05 run, retire the wait row and~~ done — routed — TODO_LIST watch row (2026-10-05)
    ~~record the runner-env proof in AGENTS.md.~~
12. ~~pdd: re-run its buildflow/flake gate once more post-cascade to confirm~~ done — routed — TODO_LIST cross-repo row (pdd pin chain)
    ~~the vendorHash settled (verified green this session, one more pass after~~
    ~~the weekly dependabot PRs land).~~

Priority 3 — user-only (blocking nothing, listed for completeness):

13. M19: enable private vulnerability reporting (Settings → Code security).
14. M20: decide the buildcache rotation policy; note it in AGENTS if it
    generalizes.
15. Confirm whether another session is intentionally experimenting with
    go-directive hand-raises (§g1) so concurrent edits stop surprising gates.

Priority 4 — optional tail (none verified-open beyond these):

16. sseparse README: link the gen procedure from the corpus doc-comment too
    (currently only `corpus.go:18-20` and README carry it — fine, but a
    cross-link from `TestCorpusJSONIsCanonical`'s failure message to the
    README section would help the next human who trips the gate).
17. example_test.go: the two new godoc examples use `_ = stream.Send...`
    discards — consider asserting output instead of printing it, so the
    examples double as tests (godoc convention permits either; note the
    tradeoff before changing).
18. scripts/smoke-examples.sh: shfmt reformatted it; consider a `--timeout`
    flag so the CI smoke job can bound total runtime explicitly.

(19–50 intentionally unused: the backlog is genuinely empty past these —
that is the point of the closeout.)

## g) Questions I CANNOT figure out myself

1. ~~**There is a concurrent actor in this repo.** Between 04:25:48 and~~ done — answered — the hand-raises were the 07:03 session's directive-floor work; kept and aligned all-three at 1.27.1 (84c3aae, 96f6b37); re-raised twice more after external normalize lowers (01cc846)
   ~~04:25:59 TODO_LIST.md changed on disk with no working-tree diff; my~~
   ~~commits after `6ca06d7` reached origin without my pushing; and the tree~~
   ~~now carries uncommitted `go 1.27` → `go 1.27.1` hand-raises in root +~~
   ~~sseparse go.mod (05:4x) that I did not author. Is another session (or~~
   ~~you) deliberately testing the directive-equality gate / directive~~
   ~~hand-raising right now? Should I treat those go.mod edits as~~
   ~~keep-and-align-all-three (ssetest too) or revert them — and should I~~
   ~~pause my own edits until that session finishes?~~
2. ~~**CI-minutes budget for the new coverage-gate job:** it installs Nix and~~ done — mooted — the Coverage gate job ran on every push since 41aac66 without budget complaints; PR restriction was never needed
   ~~runs the full test suite with coverage on every push/PR. Acceptable as~~
   ~~-is, or should it be restricted to master pushes (PRs get the plain~~
   ~~coverage job's numbers without the threshold gate)?~~
3. ~~**Release policy for tonight's changes:** GOEXPERIMENT removal and the~~ done — answered — docs-only changes rode along; the directive floor (1.27.1) is documented for the next root tag instead
   ~~new CI gates are consumer-visible only in docs (no API, no go-directive~~
   ~~change). Do you want a root `v0.6.2` cut for the docs-corrections alone,~~
   ~~or does the next API-driven release absorb them?~~
