# Status Report — 2026-10-11 10:20 — Pareto Plan Execution: Full Backlog, Waves 1–4

The 2026-10-11 SUPERB Pareto plan ran end-to-end: all 27 tasks across the
four waves. Wave 1 shipped the ssetest v0.5.0 release (tag, proxy probe,
release page) plus the two backfilled release pages; Wave 2 pinned CI
runners and mechanized two drift classes in verify.sh; Wave 3 replaced the
float coverage thresholds with an up-only ratchet and shipped the sseparse
doc touches; Wave 4 executed every cross-repo batch. Final local state:
`scripts/verify.sh` ALL CHECKS PASSED (full, including `nix flake check`),
`nix run .#coverage-gate` OK under the new ratchet, `buildflow --dry-run`
findings gate green (0 error findings, was 3), CI green on master
(38123219813).

- cover: library 100.0% (=), ssetest 100.0% (=), sseparse 100.0% (=)

## TL;DR

- ssetest v0.5.0 released: breaking `WithLastEventID(sse.EventID)` is now
  tagged, proxy-probed, and published with a release page marked Latest.
- The coverage gate is a checked-in up-only ratchet (100.0 ×3 in
  `docs/benchmarks/coverage-baseline.txt`) — zero-margin floats are gone.
- verify.sh now asserts AGENTS.md's documented facts against the tree and
  greps the bare-shfmt fingerprint; both drift classes that reddened master
  have gates instead of session vigilance.
- AGENTS.md deep-pruned 49.4 → 27.0 KB (222 → 138 lines) with every
  load-bearing rule preserved; the buildflow line-count finding is closed.
- The 2026-10-01 CI wall-clock anomaly is solved: 22 minutes of hosted-
  runner queue wait, zero retries (§a18). Post-mortem for the red streak
  in §c.

## a) FULLY DONE

| #  | Work                                                                                         | Evidence                                                                                                                                                         |
| -- | -------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | ssetest v0.5.0 docs pass                                                                     | `ssetest/README.md` zero-EventID note; `doc.go`/`options.go` confirmed current; `options_test.go` green                                                          |
| 2  | CHANGELOG cut for v0.5.0                                                                     | `[ssetest 0.5.0] - 2026-10-11` section; breaking bullet + require/directive + coverage notes                                                                     |
| 3  | Release: tag → push → proxy probe → page                                                     | `ssetest/v0.5.0` on `b7852a6`; `scripts/release-verify.sh` PASSED; proxy `.mod` inspected (go 1.27, requires v0.6.2 + sseparse v0.2.1, no replace); page live, marked Latest |
| 4  | datastartest compat check                                                                    | Their `WithLastEventID` is their own string-typed option (`datastartest/options.go:51`); ssetest breaking change invisible to them; their CI green                |
| 5  | Backfilled GitHub Release pages for v0.6.2 + sseparse/v0.2.1                                 | Both `gh release create` + `gh release list` confirm; probes passed first (guardrail #4)                                                                          |
| 6  | CI runners pinned to ubuntu-24.04                                                            | 14 `runs-on:` across 4 workflows; flip-back condition recorded in ci.yml; actionlint exit 0                                                                       |
| 7  | verify.sh AGENTS-facts assertion                                                             | Greps the ratchet-baseline + directive-policy claims vs the tree; mutation-tested red (exact fix text) and positive green; reads `coverage-baseline.txt`          |
| 8  | verify.sh bare-shfmt fingerprint guard                                                       | `\(\(![^ ]` grep over scripts/*.sh; mutation red (`((!x))` fixture), positive green; exits 1                                                                     |
| 9  | Coverage ratchet                                                                             | `flake.nix` coverage-gate reads `docs/benchmarks/coverage-baseline.txt`; below-baseline fails with fix-or-lower-deliberately message (mutation-verified), above prints ratchet-up hint (mutation-verified); `pkgs.gawk` added to runtimeInputs |
| 10 | Coverage report root scope fixed                                                             | `nix run .#coverage` measures package `.` — reports 100.0%, was ~59% diluted                                                                                      |
| 11 | sseparse raw-ID contract note                                                                | `sseparse/README.md` section; `TestCorpusJSONIsCanonical` failure message points at the ingestion procedure                                                       |
| 12 | sseparse reason comments extended                                                            | `event.go`/`corpus.go`/`reader.go` carry the zero-dep + re-export/API-break rationale above each native directive; fresh `branching-flow strong-id`: 0 rows default, 3 suppressed preserved |
| 13 | Buildflow WONT records + securitymd closure                                                  | `.buildflow.yml` policy notes (json-v1, lo.Filter); README + SECURITY.md Security Practices sections (library verified zero-logging); findings gate 3 → 0        |
| 14 | Buildflow re-run + env remainder                                                             | `buildflow --dry-run` green on the tree; oxlint `.oxlintrc.json` silences the 179 vendored-bundle warnings; `lychee`+`vulnix` in devShell; `buildflow update` ran clean (114 checks, 0 failed) |
| 15 | AGENTS deep prune                                                                            | 49.4 → 27.0 KB, 222 → 138 lines; ubuntu-24.04 pin + ratchet baseline recorded; facts assertion green against the pruned text                                      |
| 16 | release-verify release-page probe                                                            | Step 4/4 `gh release view` (skippable via `RELEASE_PAGE_CHECK=off`); verified green on the live page; CONTRIBUTING step 8 re-runs the script                      |
| 17 | CI wall-clock anomaly solved                                                                 | Run 36883370012: created 15:19:40Z, first job started 15:42:01Z — 22m21s hosted-runner queue wait; all 11 jobs `attempts=1`, distinct runners. No retries, nothing cancelled. Practical lesson: post-push CI checks must match HEAD's sha, not "newest run" |
| 18 | Red-streak post-mortem                                                                       | §c below (folded here per the TODO row's "next status report or AGENTS class note")                                                                               |
| 19 | go-datastar hygiene batch (their repo, their gates)                                          | GOEXPERIMENT removed everywhere (3 workflows + devShell + build envs + apps + AGENTS); `lint-ci` app fixed (was typechecking siblings against the root graph, exit 7 — now per-module loop); golangci pin v2.13.2 → v2.14.0 in lockstep (v2.13.2 cannot read go1.27.2 export data); directive-equality gate added; broadcast/ added to their coverage badge; two false comments corrected (nix.yml "builds from source"; root vendorHash DOES move on static/ edits — root imports `./static`); their `nix flake check` + lint-ci green; pushed `2cfd151` |
| 20 | pdd (project-discovery-daemon) pin verification                                              | Fresh clone already pins go-sse v0.6.2 (row was stale); `go mod verify` + build + tests green — row closed with evidence                                          |
| 21 | Sibling nolint migration (wise-go ×2, go-business-rules ×2)                                  | Native `//branching-flow:ignore panic` + reason lines; both repos: branching-flow 0 findings, golangci 0 issues (unknown-linter warning gone); pushed          |
| 22 | crush-config session-start CI probe                                                          | `scripts/ci-probe.sh` + `programs.crush-config.ciProbe` (on by default) + README docs; warn-only, per-repo hourly rate limit; live-tested: warns on go-datastar's 1 red run, silent on green, rate-limited; their flake check green; pushed `1ce8019` |
| 23 | branching-flow README note (native form)                                                     | Ignore-Directives section: native form preferred under golangci (evidence: sseparse); panic-limitation note updated; pushed `3ffce07d`                             |
| 24 | Upstream BD001 issue (verify-before-filing + github-voice gated)                             | Premise verified at source (`linter/brands.go:228` BasicLit-only; `repair.go:38-40` idempotence claim doesn't cover const returns); filed [go-branded-id#10](https://github.com/LarsArtmann/go-branded-id/issues/10), body verified landed (2754 bytes); draft kept in their `docs/drafts/` |
| 25 | Fleet Name-literal sweep                                                                     | wise-go 4/4 literal, cqrs-htmx 2/2, BuildFlow telemetry 4/4, browser-history 1/1 literal; go-auto-upgrade's `return toolName` is a gofinding.Detector impl, not a brand — out of BD001 scope. No sibling at risk |
| 26 | Decision briefs B1/B2/B5/B6                                                                  | `docs/brainstorming/2026-10-11_blocked-decision-briefs.md` — options + recommendation each (accept frozen red / keep manual+rule / don't gate / Option C)        |
| 27 | TODO_LIST rewritten to post-execution state                                                  | 2 trigger-gated rows + 7 Blocked remain; every executed row closed with evidence in this report                                                                   |

## b) PARTIALLY DONE

- **shfmt upstream issue: not filed, deliberately.** verify-before-filing
  Gate 2 failed — BuildFlow already fixed it (`tools/providers/
  shfmt_config.go` resolves the repo's treefmt disposition, citing go-sse's
  own ZC6 incident as motivation, and go-sse's flake `shfmt.enable = true`
  triggers the 2-space heuristic). The go-sse verify.sh bare-shfmt guard
  stays as the second net. Nothing to file.
- **crush-config probe deployment**: the feature is committed and pushed
  but reaches live sessions only after the next home-manager activation
  (owner act — `nix run` of their home switch).
- **wise-go has a pre-existing, unrelated build skew**: its `go 1.26`
  directive vs jsontext requiring go1.27 fails `go test ./...` repo-wide
  (untouched files). My migration is lint+branching-flow verified; their
  directive policy is theirs to fix.

## c) Red-streak post-mortem (2026-10-04 → 10-10) — the one paragraph

Master CI failed on ~8 pushes across six days, and every failure was one
class: **derived artifacts landing on master without their derivation
being re-run**. The Examples job failed on `templ generate -check`
(generated code stale relative to inputs); the Nix job failed on
vendorHash mismatches (go.mod/go.sum bumps — go-error-family v0.11.0 —
without recomputing the hashes); and the 10-11 tail had a non-canonical
shfmt format committed as "formatting". The agent-driven workflow edits
inputs (deps, templates, scripts) and skips the derive step because the
derivation lives in a different tool (nix, templ, treefmt) than the edit.
The fixes are now mechanized, not vigilance-based: verify.sh's tidy-clean
gate, the AGENTS-facts assertion, the bare-shfmt fingerprint guard, CI's
 Coverage gate, and the release-page probe — each drift class that reddened
master has a detector that fires before push. Residual risk: drift classes
without a detector (none currently known) and daemon commits that bypass
every local gate (mitigated by the post-sweep `go vet` rule + CI).

## d) TOTALLY FUCKED UP

1. **The daemon swept my release-prep commits three times** (81bdcec,
   bf6b586, 524f1c4 in go-sse; ba18c5f..f349988 in go-datastar): my named
   commits landed with partial file sets and heuristic-titled daemon
   commits carry the rest. Content verified complete via `git show --stat`,
   but history reads worse than the work deserves — commit surgically and
   IMMEDIATELY after each unit next time (the AGENTS rule exists; I still
   batched).
2. **Two rounds of self-inflicted shell bugs in the facts assertion**: the
   `\`` escaping inside double quotes behaves differently in this shell
   interpreter (literal backslash leak), and my quote-splice fix initially
   MISPLACED the closing backtick, producing a doubled `go go 1.27` (the
   directive variable carried the full `go 1.27` line). Two false-negative
   gate runs before the positive case passed. Lesson applied: test the
   exact construct in isolation before blaming the target file.
3. **Piped a gate through `tail` and misread its exit code** (`EXIT=0` on a
   red mutation run) — the exact trap this repo's lessons already record.
   Caught it because the FAIL message was visibly correct; re-ran unmasked.
4. **Trusted a stale memory of my own edit** when editing verify.sh the
   second time (treefmt had moved a line between my read and my edit) and
   burned an edit attempt; also let `buildflow --dry-run` confuse me for
   two runs before purging its detector cache.
5. **Almost shipped an unverified claim** ("the library never logs event
   payloads") into README's new Security Practices section — caught it in
   self-review and grepped the library before committing (true, zero
   logging). One more instance of the verify-external-claims reflex needing
   to run on MY OWN prose.

## e) WHAT WE SHOULD IMPROVE

| IMP  | Improvement (process, not product)                                                                                                        | Priority |
| ---- | ----------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| IMP1 | Commit surgically immediately per unit — the daemon race is the recurring history-quality tax                                              | high     |
| IMP2 | The briefs (B1/B2/B5/B6) need one owner sitting — they are the only remaining open work with real value                                     | med      |
| IMP3 | go-datastar's lint pin + flake golangci should get the same single-source + cross-check pair go-sse has (their pin is now workflow-single-sourced but not flake-cross-checked) | med      |
| IMP4 | verify.sh's new guards run before the expensive stages — keep that property when adding guards (cheap fail-fast ordering)                    | low      |
| IMP5 | The `RELEASE_PAGE_CHECK=off` escape hatch should be used by the next release's mid-flight probe to avoid a red herring between tag and page | low      |

## f) Up to 50 things we should get done next

1. Owner: decide B1/B2/B5/B6 from the briefs; record verdicts in
   TODO_LIST/CHANGELOG.
2. Owner: activate home-manager so the CI probe hook goes live.
3. go-branded-id: fix #10 (const resolution + repair adjacent-method
   guard); then go-sse can move `eventBrand.Name()`'s literal back behind
   a const if desired.
4. Next sseparse WPT ingestion: bump `minCorpusVectors` (29 → new size).
5. Next root tag: re-verify AGENTS directive claims (release checklist).
6. go-datastar: extend the JS-version CHANGELOG contract to the ssetest
   pins (row added to their TODO_LIST).
7. go-datastar: owner check — private vulnerability reporting setting +
   dependabot PR #23's CodeQL `analyze` failure (row added, owner-blocked).
8. wise-go: resolve their go 1.26 vs jsontext-1.27 directive skew (their
   repo; blocks their own test suite).
9. Watch the 2026-10-19 Ubuntu-26 migration date: the pin holds CI stable;
   the flip-back procedure is documented in ci.yml when nixpkgs/checkout
   stacks are verified on the new image.
10. crush-config: consider a flake cross-check for their golangci pin
    (IMP3) when someone is in that repo next.

## g) Questions I CANNOT figure out myself

None this session — the three questions open at the 05:52 report
(Ubuntu-26, v0.5.0 timing, 37-archive ruling) were resolved by execution
(pin shipped, release cut) or are now covered by the B-series briefs; the
37-archive retro-strike question remains owner-owned inside B-series
scope (no brief written — it is a one-line policy call, recommend keeping
the archives annotate-only).
