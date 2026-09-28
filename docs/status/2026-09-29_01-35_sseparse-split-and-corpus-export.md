# Status Report — 2026-09-29 01:35 — sseparse module split + corpus-as-data (go-daemon feedback)

Processed `docs/feedback/new/2026-09-29_ssetest-gaps-blocking-go-daemon-adoption.md`
end to end: extracted the parser/assertion core into a new zero-dependency
module `sseparse`, exported the WPT conformance corpus as data, promoted the
reader API to a first-class doc section, and pinned + made configurable the
1 MiB line cap. Final gates: `scripts/verify.sh` full → `ALL CHECKS PASSED`
(incl. `nix flake check`, 6 checks, all three modules); `nix run .#coverage-gate`
→ OK; buildflow full mode → all steps pass (7 pre-existing go-structure-linter
findings about the root flat layout remain, documented as deliberate).

- cover: library 99.3% (=), ssetest 98.0% (-0.4), sseparse 98.4% (new)

## TL;DR

- The go-daemon blocker is gone at HEAD: `go get .../sseparse` (once tagged) costs one module with zero third-party requires — the zero-dep property is test-guarded, not aspirational.
- The 29 conformance vectors are consumable data (`sseparse.Corpus()`), and the JSON is now the single source of truth our own tests load — parser and data cannot drift.
- Not released yet: `sseparse/v0.1.0` then `ssetest/v0.4.0` (breaking: `CodeSSEScanFailed` removed) — tag-order-sensitive, tracked in TODO_LIST.

## a) FULLY DONE

| #  | Work                                                                                                  | Evidence                                                                                                                                                                                                                                                                                                 |
| -- | ----------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | `sseparse` module: reader/event/assert/search moved, zero third-party requires, `go 1.27.1`           | `sseparse/go.mod` (no require block); `git mv` history preserved; `nix build .#checks.x86_64-linux.build-sseparse` green with `vendorHash = null`                                                                                                                                                        |
| 2  | Zero-dependency guard                                                                                 | `sseparse/go_mod_boundary_test.go` `TestModuleStaysZeroDependency` — fails on any `github.com/`/`golang.org/` line or replace directive                                                                                                                                                                  |
| 3  | Corpus-as-data export, JSON single source of truth                                                    | `sseparse/testdata/wpt_format_corpus.json` (29 vectors, machine-generated from the previous green Go literals via a one-time generator, round-trip-proven); `sseparse/corpus.go` `Corpus()`/`MustCorpus(tb)` with `ConformanceVector`/`ExpectedEvent`                                                    |
| 4  | Corpus consumers: conformance + chunk tests load from data; artifact integrity guarded                | `sseparse/wpt_format_corpus_test.go` (`TestWPTFormatCorpus`, `TestCorpusIntegrity` — floor 25, unique names, wpt:/spec:/chromium: provenance), `chunk_boundary_test.go` re-runs corpus through 1–4096B chunked readers                                                                                   |
| 5  | `WithMaxLineBytes` / `DefaultMaxLineBytes`; cap documented; boundary pinned (cap−1 parses, cap fails) | `sseparse/reader.go`; `TestReadEvents_LineCapDefault`, `TestReadEvents_WithMaxLineBytes` (lower/raise/non-positive), ReadNEvents + StreamReader variants — all wrap `bufio.ErrTooLong`, `errors.Is`-matchable                                                                                            |
| 6  | ssetest backward compatibility                                                                        | `ssetest/compat.go` re-exports the full API (aliases + wrappers, variadic `...ReadOption` additions are source-compatible); `compat_test.go` `TestCompatReExports` pins the delegation contract and keeps coverage at 98.0%                                                                              |
| 7  | errorfamily out of the parse path                                                                     | `sseparse/reader.go` wraps scan errors with `%w`; `ssetest/errors.go` deleted; `ssetest/go.mod` has go-error-family only as indirect (via go-sse). datastartest re-wraps with its own code — unaffected (verified in its `reader.go`)                                                                    |
| 8  | Trivial ask: go.mod alignment                                                                         | `ssetest/go.mod` now `go 1.27.1`; sseparse starts at `go 1.27.1`                                                                                                                                                                                                                                         |
| 9  | Reader API promoted to first-class docs                                                               | `ssetest/README.md` new "Reading from any io.Reader" section (before the HTTP table) + cap paragraph; new `sseparse/README.md` with the spec-oracle recipe; root `README.md` + `ssetest/doc.go` updated                                                                                                  |
| 10 | Module boundary guards both directions                                                                | Root `module_boundary_test.go` → `TestRootModuleDoesNotRequireSubmodules` (ssetest + sseparse); `TestModuleStaysZeroDependency` (see #2)                                                                                                                                                                 |
| 11 | Infra covers the third module                                                                         | `flake.nix`: `hermeticCheckSseparse` (`checks.build-sseparse`, `vendorHash = null`), all apps + coverage-gate (sseparse ≥ 95%); `scripts/verify.sh`; `.github/workflows/ci.yml` (test/lint/vet/coverage/vulncheck + fuzz targets relocated); dependabot `/sseparse`; `.golangci.yml` exhaustruct pattern |
| 12 | Fuzz corpus seeds moved with their targets                                                            | `sseparse/testdata/fuzz/{FuzzReadEvents,FuzzSplitSSELines}`; `FuzzWriteReadRoundTrip` seeds stay in ssetest (imports go-sse)                                                                                                                                                                             |
| 13 | golangci-lint pin skew caught and fixed                                                               | `scripts/verify.sh` cross-check failed (ci.yml v2.13.2 vs nixpkgs v2.14.0 — pre-existing drift from a flake.lock bump); all three `version:` lines bumped to v2.14.0; lint 0 issues × 3 modules                                                                                                          |
| 14 | Meta-docs                                                                                             | CHANGELOG `[Unreleased]` (Added/Changed/Removed/Fixed incl. breaking note), FEATURES.md (new sseparse section), TODO_LIST (release + adoption follow-ups), AGENTS.md (commands, architecture, gotchas, conventions), feedback file annotated processed                                                   |

## b) PARTIALLY DONE

Nothing half-shipped in-repo. The one open half is publication: the split is
code-complete and gated, but untagged. ssetest's `go.mod` carries
`require sseparse v0.1.0` + a local `replace` — resolvable in-repo today,
and at release time the tag must exist before the replace is dropped
(exact order in TODO_LIST item 1).

## c) NOT STARTED

- go-daemon's actual migration (its 10 hand-rolled framing tests →
  `sseparse.MustCorpus` vectors): deliberately not started — blocked on the
  release tags, and it is another repo's change (tracked in TODO_LIST).
- Root library v0.6.2: not needed — the root `sse` package is untouched
  (only `module_boundary_test.go`, `.golangci.yml`, README rows).

## d) TOTALLY FUCKED UP

1. First corpus JSON generation used `map[string]any` — the artifact came out
   with alphabetically sorted keys and explicit `"id": "", "retry": 0` noise
   on every event. Regenerated with tagged structs + omitempty before anything
   consumed it.
2. My boundary test flagged its own module's `module github.com/...` line as
   a third-party require (forgot to skip the `module` directive).
3. A `sed` rename (`tc.name`→`tc.Name`) over-reached into a hand-rolled local
   table in `chunk_boundary_test.go` whose fields are genuinely lowercase;
   fixed by hand, then the reverse sed over-corrected a corpus-loop line.
   Two compile cycles wasted on self-inflicted renames.
4. First cap test asserted "line exactly at cap parses" — wrong: bufio needs
   the terminator inside the cap too, so cap−1 is the largest parseable line.
   Corrected the test AND the doc comment to state the true contract.
5. An edit of mine swallowed a function header (`TestReadNEvents_ReturnsBeforeEOF`),
   leaving orphaned statements; restored it.

## e) WHAT WE SHOULD IMPROVE

| IMP  | Improvement (process, not product)                                                                                                                                                                    | Priority |
| ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| IMP1 | Corpus-as-data means hand-edited JSON is now a supported path — add a `go:generate`-style validator note (or generator) so WPT updates get notes/url citations enforced, not just the integrity floor | low      |
| IMP2 | The golangci pin skew (v2.13.2 vs v2.14.0) shipped silently via a flake-update PR — consider making the flake-update workflow run `scripts/verify.sh --fast` on its own PR                            | med      |
| IMP3 | `/mnt/buildcache` hit 100% mid-session (149 GB go-build + golangci facts caches); gates only passed with `GOCACHE=/tmp/...` overrides. Needs owner cleanup + a rotation policy                        | high     |

## f) Up to 50 things we should get done next

1. Tag `sseparse/v0.1.0` (first), then release-commit ssetest (drop replace, require resolves) and tag `ssetest/v0.4.0`; probe both with `scripts/release-verify.sh` (TODO_LIST #1).
2. go-daemon: migrate `sse_test.go` framing tests to `sseparse.MustCorpus` vectors (TODO_LIST #2).
3. go-datastar pin-bump pass now has a third module to consider when it adopts sseparse directly.
4. Clean `/mnt/buildcache` (go build cache 149 GB; golangci facts cache) and decide a rotation policy.
5. Consider teaching `docs/guides/*` about sseparse where they currently say "ssetest parser" (only README/FEATURES/AGENTS were swept this session).
6. IMP2: flake-update PRs should self-verify with `scripts/verify.sh --fast` to catch pin skews like golangci 2.13.2→2.14.0.
7. Revisit the 7 go-structure-linter findings (root flat layout) only if the project ever decides to move off the documented flat layout — they are advisory and predate this session.

## g) Questions I CANNOT figure out myself

1. `/mnt/buildcache` cleanup: it is a shared mount (other projects' caches live there) — I did not wipe 149 GB unilaterally. Owner decision on `go clean -cache` / GOMODCACHE pruning / mount resizing.

---

**Annotation 2026-09-29 01:5x (later pass, non-destructive):** row a)8
overstated the alignment fix — a subsequent `go mod tidy` (correctly) reverted
`ssetest/go.mod` to `go 1.27`: post-split, ssetest's own code no longer needs
`1.27.1` (json/v2 lives only in its tests, admitted by the `go 1.27` gate),
and tidy will always revert a hand-raised directive. sseparse stays at
`1.27.1` (it compiles json/v2). Full story in the follow-up self-review report.
