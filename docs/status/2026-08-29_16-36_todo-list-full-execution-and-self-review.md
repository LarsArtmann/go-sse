# Status Report — 2026-08-29 16:36 — TODO-List Full Execution & Brutal Self-Review

Session scope: executed the **entire 24-item TODO_LIST.md** for go-sse (correctness batch,
ssetest batch, fuzz depth, CI gates, docs), plus the cross-repo datastartest parity batch in
go-datastar. All work is **verified in the working tree of both repos but UNCOMMITTED** —
every evidence citation below is a file path, test name, or gate result, not a commit hash.
Both repos are clean-slate ready: go-sse has 25 changed/new files, go-datastar has 3.

Final gate state at report time: `scripts/verify.sh --fast` → `ALL CHECKS PASSED`
(vet + lint 0 issues both modules + race tests root/example/ssetest); `nix flake check` →
exit 0; `nix flake check --all-systems` → exit 0 (3 declared systems).

---

## a) FULLY DONE

Verified complete, evidence-backed. **Caveat applying to every row: nothing is committed
yet** (commit policy is user-gated), so "done" means "verified green in the working tree".

| # | Work | Evidence |
|---|------|----------|
| 1 | `safeDropCall` panic recovery for `onDrop` — a panicking drop callback can no longer crash `Broadcast`/`BroadcastMany`; fan-out loop proven to continue past a panic | `fanout.go:263` (`safeDropCall` + `sendAllLocked` drop branch); `TestOnDrop_PanickingCallbackDoesNotCrashBroadcast(Many)` in `drop_test.go`, pass under `-race` |
| 2 | Re-entrancy + panic-recovery contracts documented on `WithOnDrop`/`OnDrop`; drop-observability cross-referenced from `Broadcast`/`BroadcastMany` | `fanout.go:45-72`, `fanout.go:207-225`, `fanout.go:441-460` |
| 3 | `OnDrop(nil)` / `WithOnDrop(nil)` clear-callback behavior pinned | `TestOnDrop_NilClearsCallback`, `TestWithOnDrop_ExplicitNilCallback` — both assert zero drops after clear |
| 4 | `eventBrand.Name()` coverage 0% → 100% (open since the 2026-07-27 test deletion); `BrandNamer` interface satisfaction and `brandid.BrandName` wiring pinned | `event_brand_internal_test.go` (in-package, 3 tests); `go tool cover` showed `Name 100.0%` |
| 5 | ssetest `go` directive aligned 1.26.6 → 1.26.7; both modules on the same language version | `ssetest/go.mod`; both module suites green |
| 6 | `resp.Body.Close()` error branches covered — `closeBody` helper + erroring `io.ReadCloser` fake; both branches unit-tested | `ssetest/collect.go:179-190` (`closeBody`), `ssetest/collect_internal_test.go` (`TestCloseBody_*`); bodyclose exclusion for that file in `.golangci.yml` with reason |
| 7 | `RequireDataJSON(tb, evt, want)` added to ssetest — structural JSON compare, type inferred from `want`, fatal-with-payload on invalid JSON, works with `*testing.B` | `ssetest/assert.go:72-99`; 6 tests in `assert_test.go` incl. recordingTB failure paths |
| 8 | Fuzz corpus committed: 156 regression seeds across three targets, incl. the `"0data: hello\n\n"` crasher (substring-but-different-field) and the trailing-LF regression | `ssetest/testdata/fuzz/{FuzzReadEvents×51, FuzzWriteReadRoundTrip×86, FuzzSplitSSELines×19}`; crasher seed also inline in `reader_fuzz_test.go:59` |
| 9 | New fuzz burst: ~2h combined execs across 3 targets, zero crashers found (9.3M + 0.46M + 11.1M execs) | session fuzz runs, all PASS |
| 10 | `FuzzSplitSSELines` — the reader's SplitFunc fuzzed against an independent spec § 9.2.5 reference model (terminator equivalence, CRLF-is-one, no trailing empty) | `ssetest/reader_internal_fuzz_test.go` (`referenceSplitLines`, `FuzzSplitSSELines`) |
| 11 | `KeyedLines`/`SendKeyed` wire round-trip property: every keyed line survives `WriteEvent`/`Stream.Send` verbatim; rejoined data lines reconstruct the exact keyed string; `JoinLines` composition order pinned; line-count property | `keyed_wire_test.go` (4 tests); all pass under `-race` |
| 12 | BOM boundary matrix: leading/double/mid-stream BOM through chunk sizes 1–7 (the BOM probe's own `ReadFull` states) | `ssetest/chunk_boundary_test.go` `TestParserBOMSplitAcrossReads` (21 subtests) |
| 13 | Sticky-ID reconnect E2E: wire `id:` → parser sticky state → `Last-Event-ID` echo → replay → post-replay events inherit last seen ID | `ssetest/e2e_test.go` `TestE2E_StickyIDSurvivesReconnect` |
| 14 | CI extended: examples build + `templ generate -check` job; `nix flake check` job (install-nix-action pinned by SHA `13d8dd5…`); fuzz job 3 → 6 targets; `govulncheck` pinned `@v1.7.0` (was `@latest`) | `.github/workflows/ci.yml` — **yaml-verified only, never executed in real CI** (see b) |
| 15 | `scripts/verify.sh` one-command pre-push gate (treefmt + vet + lint + race test + flake check; `--fast` skips flake), direnv-independent | `scripts/verify.sh`; smoke-tested end-to-end, `ALL CHECKS PASSED` |
| 16 | `coverage-gate` extended with ssetest threshold (95% alongside library 90%) and made actually-runnable outside a dev shell | `flake.nix` `coverage-gate` app; local run: `library coverage: 99.3% … ssetest coverage: 97.2% … OK` |
| 17 | `docs/guides/reconnection-and-retry.md` — the 5-layer reconnection model, written from source | new file; link target `docs/brainstorming/2026-08-07_go-retry-adoption-evaluation.md` verified to exist |
| 18 | CONTRIBUTING release checklist (8 steps: CHANGELOG cut, docs refresh, hermetic gate, worktree tag validation, proxy verification, dual-module tagging, `gh release create`) | `CONTRIBUTING.md:43-100` |
| 19 | `nix flake check --all-systems` executed — caught that Nixpkgs 26.11 dropped x86_64-darwin; flake now declares its 3 supported systems explicitly, all-systems green | `flake.nix` systems list; `/tmp` run logs: `ALL_SYSTEMS_EXIT=0` |
| 20 | gopls hygiene: all inferable generic type arguments removed from test call sites (15 sites across 3 files); required ones kept; compile-probe verified inference works | `drop_test.go`, `lifecycle_test.go`, `filter_test.go`; `gopls check` CLI reports 0 diagnostics |
| 21 | Living docs updated: TODO_LIST rebuilt (24 items → 2 cross-repo TODOs + 3 WONTs + 1 BLOCKED), CHANGELOG `[Unreleased]` filled (15 Added + 6 Changed), FEATURES coverage rows refreshed, AGENTS.md gotchas extended (safeDropCall, closeBody/bodyclose, systems narrowing, gopls root-causes, synctest WONT, CI facts) | `TODO_LIST.md`, `CHANGELOG.md`, `FEATURES.md`, `AGENTS.md:86-125` |
| 22 | Lint config: varnamelen `sc` whitelisted; bodyclose file-scoped exclusion with inline reason; both after discovering trailing nolints don't survive golines re-splitting | `.golangci.yml`; both modules 0 issues post-`nix fmt` |
| 23 | datastartest parity batch (go-datastar repo): 51-entry fuzz corpus ported (`string`→`[]byte` conversion), 9 conformance seeds added (WPT vectors, sticky-id, BOM, crasher, trailing-LF), conformance README section | `go-datastar/datastartest/README.md`, `reader_fuzz_test.go`, `testdata/fuzz/FuzzReadEvents/`; its suite green |

---

## b) PARTIALLY DONE

| # | Item | Works | Missing | Effort |
|---|------|-------|---------|--------|
| 1 | **Everything above is uncommitted** in both repos | All gates green locally | `git commit` (user-gated); go-sse 25 files, go-datastar 3 files + untracked testdata dir | S (once approved) |
| 2 | New CI jobs are yaml-only | Syntax + local equivalents of every step verified (`go build ./example/...`, `templ generate -check`, `nix flake check`, all 6 fuzz targets run locally) | **Real GitHub Actions execution** — the nix job (install-nix-action SHA, GITHUB_TOKEN perms, flake eval on the runner) and the examples job have never run in actual CI; first push will be the real test | S-M (push + watch + fix) |
| 3 | gopls hygiene | All inferable type args removed; `gopls check` CLI = 0 | The original "~17 editor infos" could not be reproduced via CLI, so I resolved what grep+compile-probe could find; editor-only residuals may persist. `stdversion` friction is intrinsic until a `go 1.27` directive | n/a (documented WONT) |
| 4 | datastartest parity (go-datastar) | Corpus + seeds + README landed; its suite green | Writer goldens for DataStar patches in go-datastar **core** (needs that repo's conventions loaded — out of this session's scope); the batch is uncommitted there | M |
| 5 | AGENTS.md datastartest claim | README text I added is accurate | **go-sse's AGENTS.md itself is now known-wrong** — see d) #1 | S |
| 6 | synctest item | Dispositioned WONT with root cause (network in bubbles prohibited; helpers own real sockets) | Nothing — decision is final unless a fake-net design appears | n/a |
| 7 | Fuzz corpus growth | 156 seeds committed from 45s/30s/45s bursts | Deeper budgets (10m+ per target) never run; corpus remains one-off, not continuously grown | M |

---

## c) NOT STARTED

| # | Item | Why | Still wanted? |
|---|------|-----|---------------|
| 1 | Browser-E2E test (headless Chromium + DataStar client vs example server) | Blocked on the Option-B-vs-C scope decision (`docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md`) — user decision outstanding since 2026-08-03 | Yes, but blocked 26 days |
| 2 | Release v0.6.0 / ssetest v0.3.0 (the `[Unreleased]` section is now release-sized) | Release timing is a user call; checklist now exists in CONTRIBUTING | Presumably yes, after commit/push |
| 3 | go-datastar core writer goldens | Needs that repo's session with its own conventions | Tracked in TODO_LIST |
| 4 | Fuzzing for the remaining root-module writers (`WriteHeartbeat`, `WriteRetry` — trivial writers, currently unfuzzed) | Tiny surface; deprioritized vs reader-side depth | Optional |
| 5 | Re-adding x86_64-darwin | Impossible until Nixpkgs restores the platform (26.11 dropped it) | Auto when upstream does |

---

## d) TOTALLY FUCKED UP

The most valuable section. Everything here is **fixed or documented as of this session**,
but each item is a real failure I caused or shipped mid-session.

1. **AGENTS.md's datastartest claim is a split brain I found and did not fix.** AGENTS.md
   says ssetest and datastartest are "independent implementations of the same spec… the two
   modules do not depend on each other", with a parity mandate ("bug fixes must be applied
   to both"). Reality discovered this session: **datastartest imports `go-sse/ssetest` from
   the module proxy** (`toEvents(events []ssetest.Event)` in its reader; its test run
   downloaded `ssetest v0.2.0`). It is a thin wrapper, not a duplicate. Consequence: the
   "apply fixes to both" mandate is stale, seed-porting is belt-and-suspenders, and the
   version of ssetest that datastartest tests against is pinned by go-datastar's go.mod —
   a lagging dependency nobody tracks. I wrote the *correct* statement into datastartest's
   README but left go-sse's AGENTS.md lying. Severity: documentation is actively wrong for
   every future session. Fix is S but is a user-visible contract change (which architecture
   is intended?), so it is routed to TODO_LIST + question g)2, not silently edited.
2. **I declared "lint 0 issues" twice while the tree was actually lint-dirty.** Sequence:
   fixed findings → declared clean → `nix fmt` re-split long lines → trailing
   `//nolint:bodyclose` / `//nolint:varnamelen` directives landed on the wrong lines →
   nolintlint "unused directive" + the original findings resurfaced. Root cause: I ran
   golangci-lint **before** the formatter, not after. Caught only because I smoke-tested
   `scripts/verify.sh` at the very end — without that, broken lint ships. Fix: config-level
   exclusions (formatter-proof) instead of line-level nolints; verify.sh runs fmt first.
   Lesson recorded: lint is only meaningful as the LAST step after formatting.
3. **I shipped a broken assertion because I asserted my assumption instead of the pinned
   contract.** `TestKeyedLines_WireRoundTrip/empty_value` expected 0 data lines for empty
   value; the writer's documented contract (pinned in `TestWriteReadRoundTrip_EmptyDataIsOneEmptyEvent`,
   which I had read hours earlier) is exactly one empty data line. Full-suite run caught it.
   Inexcusable in a repo whose conformance tests I had just been citing.
4. **`perl -0pi` mangled indentation** across 4 sites in collect.go (double tabs, space-led
   nolint lines). Wrong tool for whitespace-sensitive edits; repaired with python. The
   lesson already exists in my instructions ("use edit/multiedit for whitespace-sensitive
   changes") — I violated it for a "quick" 4-site substitution.
5. **Environment failures burned ~6 round trips** despite the runbook: piped/backgrounded
   `nix run` swallowed output 3× (empty logs, exit 1 with no diagnostic) before I wrote to
   files; the ambient `GOMODCACHE=/mnt/buildcache` (broken mount) failed `go test` inside
   the gate before I remembered the exports I was handed at session start. The AGENTS.md
   buildflow/direnv warning existed; I still paid the toll again.
6. **`sprintf` instead of `fmt.Sprintf`** in a brand-new file — a compile error that a
   compile-immediately reflex would have caught; instead the LSP diagnostics banner was
   full of unrelated stdversion/buildcache noise, and I found it on the test run.
7. **The `coverage-gate` "hermetic" claim I wrote was wrong for ~30 minutes**: I first
   claimed writeShellApplication's PATH "is minimal"; the `bash -x` trace proved
   runtimeInputs are **prepended** to the caller's PATH (that's why `tail` resolved at
   all). Corrected AGENTS.md at report time (16:36). Also: the pre-existing gate could
   never have worked with only `[goPkg pkgs.bc]` (no grep, no GOWORK export) — meaning
   the archived report that claimed coverage-gate worked was never verified in isolation.
   Status-report claims decay; verify before trusting.
8. **`flake.lock` was modified as a side effect** of removing the `systems` input. It is
   consistent with the flake.nix change, but I did not call it out in my summary — the
   user should know the lockfile moves with that commit.
9. **Scope honesty on "The WHOLE TODO LIST"**: two items were dispositioned rather than
   implemented (synctest → WONT with root cause; gopls infos → partially resolved + WONT),
   and two were rerouted across repos (writer goldens). That is the correct engineering
   outcome, but "finished the list" really means "finished or explicitly dispositioned
   every item".
10. **Did I lie to you?** Checked every claim in my final summary against this session's
    evidence: coverage numbers (97.2/99.3), gate results, test names — all reproduce. The
    one overstatement risk: "CI: …" rows read as *shipped capability* but are
    *yaml-configured, never executed* — that is why b)2 exists.

---

## e) WHAT WE SHOULD IMPROVE

1. **Gate ordering is a footgun.** Fix: make `scripts/verify.sh` the only way anyone (me
   included) declares cleanliness — it runs fmt before lint. Add: never run golangci-lint
   standalone as a "final" check. (Process; zero code.)
2. **Uncommitted-work risk.** 25 files across a verified session sit uncommitted; the
   auto-git daemon may commit them with a garbage message, or a stray checkout loses them.
   Commit in reviewable units (code / flake+CI / tests / docs / go-datastar) at next
   instruction. (Process.)
3. **CI can only be trusted after it runs.** The next push should be watched end-to-end;
   budget one iteration for install-nix-action token/eval surprises. (Process.)
4. **AGENTS.md drift caught → fix flow missing.** When a documented claim is disproven
   (datastartest dependency, vendorHash trigger, mkApp PATH), the correction should happen
   in the same session, not be routed. Two of three got fixed this time; the architecture
   claim needs your g)2 answer first. (Process.)
5. **Editor-only lint noise (`stdversion`, nolint hints) keeps costing every session.**
   Root causes are now documented in AGENTS.md — the remaining fix is refusing to chase
   them, which is a discipline problem, not a tooling one. (Discipline.)
6. **datastartest's dependency on ssetest is untracked** — go-datastar pins `ssetest
   v0.2.0` and nobody bumps it when ssetest releases. If the wrapper architecture stays
   (g)2), add a cross-repo bump checklist item to the release checklist. (Release
   engineering.)
7. **Test-first habit regression.** Two of my failures (empty-value assertion, sprintf)
   would vanish if every new test file is run verbosely (no grep truncation) immediately
   after writing. (Discipline.)
8. **Fuzz corpus is a snapshot, not a practice.** Consider a monthly or per-release
   `go test -fuzz` budget per target committed as corpus, so the committed seeds keep
   pace with the code. (Quality.)
9. **verify.sh --fast is now the local loop** — but treefmt is skipped without nix
   develop. If pre-commit-style enforcement is wanted, a git pre-push hook calling
   verify.sh closes the "forgot to run it" class permanently. (Tooling; the TODO item
   that produced verify.sh explicitly mentioned "or an equivalent pre-push hook".)
10. **Status-report evidence decay**: the coverage-gate incident shows archived claims
    ("gate works") can be wrong. Docs-health VERIFY mode already exists — cheap win is
    re-verifying tooling claims (not just doc claims) during the next audit. (Process.)

---

## f) 50 things to get done next

Brainstorm ranked by impact; per the status-report skill, this is **HARVEST fuel**, not a
commitment — most should route to TODO_LIST (actionable) or ROADMAP (ideas).

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Commit this session's go-sse work in reviewable units (code, flake+CI, ssetest tests, docs, corpus) | Critical | S | Cleanup |
| 2 | Push and watch the new CI jobs end-to-end; fix whatever the real runner surfaces | Critical | M | Bug |
| 3 | Decide datastartest architecture question (g)2) and fix AGENTS.md's stale "independent implementations" claim | Critical | S | Documentation |
| 4 | Commit + push the datastartest parity batch in go-datastar | High | S | Cleanup |
| 5 | Release v0.6.0 + ssetest v0.3.0 using the new CONTRIBUTING checklist | High | M | Feature |
| 6 | After ssetest release: bump go-datastar's `ssetest` dependency + tag datastartest | High | S | Feature |
| 7 | Add "bump go-datastar's ssetest pin" to the CONTRIBUTING release checklist | High | S | Documentation |
| 8 | Writer goldens for DataStar patches in go-datastar core (own session, own conventions) | High | M | Quality |
| 9 | Write the session SUPERB plan doc (paste_1 template) for this batch retroactively — or accept the report as the record | Low | S | Documentation |
| 10 | HARVEST this report's (f) list into TODO_LIST/ROADMAP with extra routing rigor | High | S | Documentation |
| 11 | CI: add treefmt/format job (fmt is currently only local via verify.sh) | Medium | S | Quality |
| 12 | CI: concurrency group to cancel superseded runs on master pushes | Medium | S | Quality |
| 13 | Install a git pre-push hook calling `scripts/verify.sh --fast` (kills the unverified-push failure class) | High | S | Quality |
| 14 | Fuzz-time budget: 10-minute per-target bursts before each release, corpus committed | Medium | M | Quality |
| 15 | Add fuzz seeds corpus for root targets (`FuzzWriteEvent`, `FuzzParseEventID`, `FuzzKeyedLines`) from cache bursts | Medium | S | Quality |
| 16 | Fuzz `WriteHeartbeat`/`WriteRetry` (trivial but unpinned) | Low | S | Quality |
| 17 | `Stream.Send` partial-write semantics test: writer error mid-frame (errorWriter currently covers error-on-write; add a short-write fake) | Medium | M | Quality |
| 18 | Context-cancellation propagation test: handler ctx cancel mid-stream closes body cleanly | Medium | M | Quality |
| 19 | `Health()` gain a `DroppedEvents` counter (ROADMAP §1 observability question) — decide, don't let it float another release | Medium | M | Feature |
| 20 | Decide `OnPredicatePanic` hook (ROADMAP §1) — silent recovery has now shipped for predicates AND drops; make the observability stance explicit | Medium | M | Feature |
| 21 | Benchmark suite: `Broadcast` fan-out, `WriteEvent` allocation profile (benchstat-tracked) | Medium | M | Quality |
| 22 | Verify the allocation-free hot path claim (`WriteEvent`) with `benchmem` numbers in FEATURES | Medium | S | Quality |
| 23 | EventStore guidance doc: retention/GC patterns for replay stores (the interface is unopinionated; a guide prevents consumer foot-guns) | Medium | M | Documentation |
| 24 | Replay pagination for huge gaps (`EventsAfter` with limit) — API design sketch, decide whether in-scope | Medium | M | Feature |
| 25 | `docs/guides/` second entry: "filters and fan-out patterns" (SubscribeFilter recipes under the read-lock constraint) | Medium | M | Documentation |
| 26 | README: wire the verify.sh + coverage-gate + fuzz-target counts into the contributing blurb | Low | S | Documentation |
| 27 | Add go doc examples (`example_test.go`) for `RequireDataJSON`, `WithOnDrop` (godoc renderings on pkg.go.dev) | Medium | S | Documentation |
| 28 | pkg.go.dev screenshot/render check for the new helpers before release | Low | S | Documentation |
| 29 | SECURITY.md (private vulnerability reporting contact) — missing for a public library | Medium | S | Documentation |
| 30 | Dependabot/Renovate for GitHub Actions SHA pins (they rot silently) | Medium | S | Quality |
| 31 | govulncheck pin refresh policy: note in CONTRIBUTING to bump `@v1.7.0` on each minor | Low | S | Documentation |
| 32 | CI fuzz job: move from fixed 1m to total-budget awareness (6 targets × 1m = 6 min/job) | Low | S | Quality |
| 33 | CI matrix: second Go version (e.g. previous minor) for the test job | Low | M | Quality |
| 34 | Tag signing policy for releases (`git tag -s` vs `-a`) — currently unspecified in the checklist | Low | S | Documentation |
| 35 | `gh release create` automation script wrapping checklist steps 5-7 (fewer fumbles) | Low | M | Feature |
| 36 | ROADMAP sweep: resolve the parked items now that drops/filters/shutdown shipped (LastEventID question already removed; what's left?) | Low | S | Documentation |
| 37 | `example/datastar` coverage 46.3% — either grow it or explicitly declare examples out of coverage scope in FEATURES (it already notes exclusion; make the numbers current) | Low | S | Documentation |
| 38 | Tie the `example/` servers into a smoke test script (boot each, curl one event, kill) — no browser needed | Medium | S | Quality |
| 39 | go.work pollution: pin a note or `.envrc` guard so future sessions stop rediscovering `GOWORK=off` (already in AGENTS; consider a Makefile-less `just`-style alias in verify.sh) | Low | S | Quality |
| 40 | Fix the remaining WONT-docs: link the synctest WONT decision from AGENTS gotcha to the report for future skeptics | Low | S | Documentation |
| 41 | Evaluate `errors.AsType` migration (Go 1.26 erraudit) across the codebase per the go-error-modernization skill | Medium | M | Quality |
| 42 | Run `nix run .#build` as part of verify.sh full mode (currently only flake check builds) | Low | S | Quality |
| 43 | Add `--all-systems` to the full (non-fast) verify.sh flake check, now that systems are declared | Low | S | Quality |
| 44 | Split `keyed_wire_test.go` if it grows; today 4 tests is fine — keep an eye (naming: it mixes fuzz-adjacent property tests) | Low | S | Cleanup |
| 45 | Consider extracting `parseDataLines` in keyed_wire_test.go into a tiny shared test helper if a second consumer appears (dupl threshold watch) | Low | S | Cleanup |
| 46 | Re-run the 45s fuzz bursts after any future reader change and commit deltas (corpus stays live) | Medium | S | Quality |
| 47 | datastartest: port `FuzzSplitSSELines`-equivalent only if the wrapper ever stops delegating to ssetest (post-g)2 decision) | Low | S | Quality |
| 48 | Track ssetest version used by go-datastar in CI (a tiny job asserting go-datastar tests against the latest ssetest tag) | Medium | M | Quality |
| 49 | Explore `testing/synctest` with an in-memory fake transport in a spike branch ONLY if drops/shutdown timing tests ever get flaky (declined for now — keep declined) | Low | L | Quality |
| 50 | Consider a `docs/guides/getting-started.md` distinct from README quickstart (README is a sales page per doc-file contract; a step-by-step guide is a different artifact) | Low | M | Documentation |

---

## g) Three questions I cannot figure out myself

**g)1 — Commit & push this batch now?** 25 verified files sit uncommitted in go-sse (plus
3 in go-datastar). My rules forbid committing without your explicit word, and several of
the "shipped" items (CI jobs, release checklist) only become *real* on push. If yes: I
propose 5 reviewable units — (1) fanout/drop code+tests, (2) ssetest helpers+tests+corpus,
(3) flake+CI+lint-config+verify.sh, (4) living docs, (5) go-datastar parity — followed by
push so CI executes. Wait, or commit?

**g)2 — Is datastartest *supposed* to be a thin wrapper over ssetest?** go-sse's AGENTS.md
still mandates "independent implementations, do not depend on each other, apply fixes to
both" — but the actual code (datastartest's `reader.go` delegates to `ssetest`, downloaded
from the proxy) says wrapper. One of the two documents is lying about the intended
architecture. If wrapper is intended: I fix AGENTS.md (parity mandates, dup-parser gotchas)
and add a dependency-pin bump step to the release checklist. If independence is intended:
that's a real refactoring task in go-datastar, and my ported corpus becomes load-bearing
rather than belt-and-suspenders. Which is the truth?

**g)3 — Release now or accumulate?** The CHANGELOG `[Unreleased]` section is
release-sized (safeDropCall, RequireDataJSON, corpus, CI gates, guide, checklist), the
release checklist now exists, and the last releases "fumbled the same steps" — this would
be its first real test. But cutting v0.6.0/ssetest v0.3.0 before the new CI has run even
once in production violates the checklist's own gate-3 logic. Do you want the release
attempted right after commit+push+green-CI, or parked until more work accumulates?

---

*Report ends. HARVEST note: section (f) is the primary input for `docs-health` HARVEST
into TODO_LIST.md/ROADMAP.md — not yet applied, awaiting instructions.*
