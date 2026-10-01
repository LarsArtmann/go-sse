# Status Report — 2026-10-01 15:05 — Directive-floor restore + audit closeout

Scope: restore the `go 1.27.1` floor after another post-tag normalize
lowering, then close the 2026-09-29 audit's remaining engineering items
(coverage floors, gen_corpus round-trip test, GOEXPERIMENT guard, go.sum
boundary coverage, conformance doc, path-filter audit, fuzz soak). Final
gate state: `scripts/verify.sh` ALL CHECKS PASSED (incl. tidy + directive
gates, lint ×3 modules, race tests ×3 modules, `nix flake check`); the
fresh 6×5-minute local fuzz soak is clean on the dependency-bumped tree.

- cover: library 99.3% (=), ssetest 100.0% (=), sseparse 99.2% (=)

## TL;DR

- Daemon commit `73f987d` re-lowered root+sseparse to `go 1.27` (third
  normalize regression; this one post-tag) while also landing the legit
  sseparse v0.2.0 pin + vendorHash repair — the pin was kept, the lowering
  re-raised, and the full gate re-verified green.
- Five audit items closed with tests/gates, one with an audit finding
  (no `paths:` filters exist in this repo's workflows).
- The local fuzz soak exposed that _my own soak script_ used the stale
  module mapping; CI was already correct. Re-soaked the two sseparse
  targets properly.

## a) FULLY DONE

| #  | Work                                                                                    | Evidence                                                                                                                                                                                                                                                                                                       |
| -- | --------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | `go 1.27.1` floor re-restored in root + sseparse (post-tag normalize lowering reverted) | `git diff 73f987d` on `go.mod`/`sseparse/go.mod`; verify.sh directive-equality gate green; ssetest's legit sseparse v0.2.0 pin + recomputed `vendorHashSsetest` kept                                                                                                                                           |
| 2  | Coverage-gate floors raised to measured levels                                          | `flake.nix` `coverage-gate` app now 99/100/99 (was 90/95/95); `nix run .#coverage-gate` prints `99.3 / 100.0 / 99.2 — OK`; measured independently via `go test -cover` in all three modules                                                                                                                    |
| 3  | gen_corpus ingestion contract pinned by a process-level round-trip test                 | `sseparse/gen_corpus_roundtrip_test.go`: 5 scenarios (idempotence, ingestion, dup-skip, dup-drift abort, dispatch-mismatch abort) against real `go run gen_corpus.go` sandboxes; `-race -count=1` green 4×; `golangci-lint` 0 issues; mutation of the shared-corpus slice caught and fixed with `slices.Clone` |
| 4  | GOEXPERIMENT regression guard in verify.sh                                              | `scripts/verify.sh` must-find-nothing grep over flake.nix/scripts/workflows (self-match excluded via `--exclude=verify.sh`); guard verified red-then-green during authoring; full gate green                                                                                                                   |
| 5  | Module-boundary test also fails on submodule paths in `go.sum`                          | `module_boundary_test.go` scans both files; mutation-verified (injected `sseparse v9.9.9` go.sum line → test fails with the new message; clean tree passes)                                                                                                                                                    |
| 6  | Living conformance doc distilled                                                        | `docs/conformance.md` (three surfaces, corpus contract, D1–D6 found-and-fixed with pinning tests, extension procedure); linked from AGENTS.md's parser-corpus gotcha; all links + test names verified against the tree                                                                                         |
| 7  | Path-filter audit (this repo's half)                                                    | `grep` over all four workflows: none carries a `paths:` filter (ci.yml pushes+PRs, actionlint explicitly all-pushes, two scheduled) — the silent-skip class that hit go-datastar is structurally absent here; recorded in the TODO cross-repo row                                                              |
| 8  | Local 5-min fuzz soak on the dependency-bumped tree                                     | 6 targets × 5m, rc=0: FuzzWriteEvent 52.9M execs, FuzzParseEventID 55.4M, FuzzKeyedLines 50.9M, FuzzReadEvents 97.0M (sseparse), FuzzWriteReadRoundTrip 55.5M, FuzzSplitSSELines 140.3M (sseparse); no crashes, no new testdata/fuzz failures                                                                  |
| 9  | Docs maintenance                                                                        | `TODO_LIST.md` rewritten (5 items closed, 2 trigger-gated kept), `CHANGELOG.md` `[Unreleased]` gained 6 lines, AGENTS.md directive-dance note updated with the post-tag instance                                                                                                                               |
| 10 | Cross-repo claims re-verified                                                           | go-datastar: master CI green 2026-10-01, `GOEXPERIMENT: jsonv2` still at their ci.yml:53 (batch still valid); pdd: not cloned under ~/projects — noted in the TODO row                                                                                                                                         |
| 11 | Fuzz-target existence guards in all three modules (IMP1, implemented same-session)      | `fuzz_guard_test.go` ×3 (root: WriteEvent/ParseEventID/KeyedLines; sseparse: ReadEvents/SplitSSELines; ssetest: WriteReadRoundTrip); mutation-verified (renamed `FuzzKeyedLines` → guard red with actionable message → restored green); lint 0 issues ×3                                                       |

## b) PARTIALLY DONE

None — every item started this session shipped completely.

## c) NOT STARTED

- **go-datastar hygiene batch** (~14 sub-items) and the **pdd pin bump**:
  deliberately out of scope — both are separate repos with their own gates
  and the TODO frames the batch as one sweep in its own repo; pdd is not
  cloned locally. Claims re-verified current instead (row #10).

## d) TOTALLY FUCKED UP

1. The soak script fuzzed `FuzzReadEvents`/`FuzzSplitSSELines` in **ssetest**
   from stale module memory — both moved to sseparse in the split, so two
   targets ran as green no-ops ("warning: no fuzz tests to fuzz"). Caught by
   reading the soak output; re-soaked in sseparse. ~10 minutes wasted, and
   the first "clean soak" announcement would have been half vacuous.
2. The round-trip test's mismatch scenario mutated `Events[0]` through the
   shared backing slice while the parallel ingests scenario read the same
   vector — a real data race that also made results scheduling-dependent.
   Caught by running the suite 3× under `-race` before committing; fixed
   with `slices.Clone` per derived vector.
3. The first directive fix was a silent no-op: `sed -n '2s/...'` targeted
   line 2, but these go.mod files carry a blank line after `module`, so the
   directive sits on line 3. Caught by re-reading the file immediately
   after the edit.
4. The gen_corpus sandbox needed three iterations to compile (missing
   go.mod → missing package sources → corpus/pending under `testdata/` for
   the go:embed and the generator's relative paths). Lesson applied: mirror
   the real layout exactly instead of approximating it.

## e) WHAT WE SHOULD IMPROVE

| IMP  | Improvement (process, not product)                                                                                                     | Priority |
| ---- | -------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| IMP1 | RESOLVED same-session: the fuzz-target existence guard now lives in all three modules (row #11).                                       | —        |
| IMP2 | Ad-hoc soak/verification scripts must derive module→target mappings from the tree (`grep -rn "^func Fuzz"`), never from AGENTS memory. | med      |

## f) Up to 50 things we should get done next

1. The cross-repo batch and pdd bump stay as the next-session sweep (see
   TODO_LIST Cross-repo). IMP1's guard shipped this session; nothing new
   harvested beyond the standing rows.

## g) Questions I CANNOT figure out myself

None this session.
