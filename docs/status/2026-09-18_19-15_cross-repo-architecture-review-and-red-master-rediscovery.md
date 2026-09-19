# Status Report — 2026-09-18 19:15 — Cross-Repo Architecture Review (go-etag Lessons) + Red-Master Rediscovery

**Session scope:** read-only analysis session, zero code mutations authored. Three questions
answered: why go-sse keeps its Go files in root (documented parked split), a critique of
ROADMAP.md (4 issues found), and what go-sse can learn from go-etag (5 lessons, 2 of which are
reference implementations for go-sse's own parked ideas). Mid-session the repo's gate state was
audited because this report mandates fresh measurements — that audit rediscovered that **master
CI has been red for 5 days** and diagnosed today's regression wave (commit `531cb96`, authored
outside this session, committed + pushed by the auto-daemon at 19:04 today).

Final gate state at report time: `scripts/verify.sh` **not runnable** (see d1/d2 — the tree
requires `go >= 1.27.1`, local toolchain is go1.26.7 with `GOTOOLCHAIN=local`); `go vet ./...`
fails with the toolchain error; `nix run .#coverage-gate` **exits 1 with a 0-byte log** (twice,
with sane caches — see d3); `origin/master == HEAD` (`531cb96`); master CI red on the last
three pushes (2026-09-13, 09-15, 09-18).

- cover: library N/A (+Δ n/a), ssetest N/A (+Δ n/a) — measurement impossible this session (d1/d3); baseline for the next measurement is the 2026-09-04 report's line. Per convention no stale number is quoted as current.

## TL;DR

- Read-only analysis session delivered: root-layout answer (parked split, 7 non-test files ≈ 1.9k lines), 4 ROADMAP defects, 5 go-etag lessons — 2 of them direct reference implementations for go-sse's parked ideas (alias-shim split playbook; typed error-code surface).
- **Master CI red ×3 consecutive pushes since 2026-09-13** (Lint-only at first; today Vet + Lint + Nix flake check all red).
- Today's wave: ungated 6-file bump commit `531cb96` (go directive 1.26.7→1.27.1, go-branded-id v0.5.1→v0.6.0, go-error-family v0.10.0→v0.10.1, `exhaustruct`→`exhaustruct_v5`, flake.lock) — violates the documented vendorHash rule; no gate ran before push.
- Every local gate is currently un-runnable; coverage is unmeasurable (d1/d3), so this report carries no fresh coverage number by design.
- Two self-caught honesty defects in this session (d5): an unverified docs-gap claim (corrected by verification) and a pipeline-masking near-miss while running vet.

## a) FULLY DONE

| # | Work                                                                                                                                                                                                                                                                                               | Evidence                                                                                                                                                                                                                                                                                                                                           |
| - | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Root-file-layout question answered with receipts: single-package flat layout is deliberate, split is parked with explicit re-open triggers                                                                                                                                                         | `AGENTS.md` architecture table (1 layer = 1 file); `wc -l *.go` → 7 non-test files, 6,389 total lines incl. tests; ROADMAP.md:62 parked decision (deferred 2026-07-25, 4 triggers)                                                                                                                                                                 |
| 2 | ROADMAP.md critique delivered: 4 concrete defects                                                                                                                                                                                                                                                  | (1) §1 exit criteria met (ROADMAP.md:26) but sequencing table still points "Now" at it (:11); (2) §2 Redis EventStore idea (:45) is a parked-split trigger (:62) without a cross-link; (3) unexplained "CLTY" token (:51); (4) 2-of-4 wire-only consumer count is 2 months stale                                                                   |
| 3 | go-etag lessons mined: alias-shim migration playbook (`deprecated.go` + `deprecated_test.go` export-parity suite + tombstone `doc.go`, removed at v1.0.0) and typed error-code reference (`server/code.go`: `Code`, family constructors, declarative `errorTemplates`, bidirectional pinning test) | go-etag AGENTS.md:76-93, 160; both map 1:1 onto go-sse ROADMAP.md:62 (parked split) and :81 (typed-code raw idea)                                                                                                                                                                                                                                  |
| 4 | CLTY provenance diagnosed: token was born garbled and propagated by harvest for ~2 months                                                                                                                                                                                                          | `git log -S CLTY -- ROADMAP.md` → introduced in initial ROADMAP commit `5f2ba13`; copied verbatim through ≥9 archived status reports (2026-07-23 → 2026-08-03 tables) without ever being decoded                                                                                                                                                   |
| 5 | Session's "no WHATWG §9.2 conformance account" claim verified and corrected                                                                                                                                                                                                                        | grep docs/ → full conformance program EXISTS: `docs/planning/archived/2026-08-16_09-21_SUPERB-spec-based-hardcore-sse-conformance-testing.md` (D1–D6 deviations table), execution + closeout reports, WPT corpus tests, AGENTS.md parser-contract gotchas. Residual gap is narrower: no LIVING conformance account (material is archived)          |
| 6 | Red master rediscovered and root-caused to commit level                                                                                                                                                                                                                                            | `gh run list --workflow ci.yml` → green 2026-09-10 (`34543465191`), red 09-13 (`34749630537`, Lint only), red 09-15 (`35000099306`, Lint only), red today (`35372363659`, Vet + Lint + Nix flake check); `git show 531cb96` diff inspected (see d2); local repro: `go vet` → `go.mod requires go >= 1.27.1 (running go 1.26.7; GOTOOLCHAIN=local)` |
| 7 | Report + mandatory freshness checks: origin sync, CI history, coverage-gate attempts (2, honest failures)                                                                                                                                                                                          | `git log origin/master -3` == HEAD; coverage-gate log captured to `/tmp/covgate.log` (0 bytes, REAL_EXIT=1, verified not a pipeline artifact)                                                                                                                                                                                                      |

## b) PARTIALLY DONE

1. **Coverage measurement** — attempted twice: ambient caches (`/mnt/buildcache` exists and
   looks valid) then explicit `/tmp` caches. Both exit 1; second run's full log is 0 bytes, so
   the underlying failure is swallowed by the app (d3). What remains: re-run the gate after d1
   is resolved and record a real number in the next report.
2. **Durable documentation of the go-etag lessons** — content fully worked out (a3, a5) and
   insertion points identified (ROADMAP.md:62 and :81), but zero living docs were edited this
   session per the explicit "THEN WAIT FOR INSTRUCTIONS" order. f7–f16 are the concrete edits.
3. **ROADMAP fixes** — all four defects precisely specified (a2), none applied (same wait order).
4. **`scripts/verify.sh` gate** — intentionally not run: the tree cannot compile under the
   local toolchain (d1). Running it would produce a foregone-conclusion failure, not information.

## c) NOT STARTED

1. **Fixing master** — root cause fully diagnosed (d1/d2), fix not started: keep-vs-revert of
   the bump is a user decision (g1) because the bump was authored outside this session.
2. **HARVEST of this report's section f into TODO_LIST/ROADMAP** — deferred by the explicit
   wait order; per docs/status/AGENTS.md this is the post-report step, still pending.
3. **Wire-only consumer recount** (f13) — not started; requires cross-repo access or user knowledge (g3).
4. **Bounding the three §1 leftover design questions** (backpressure policy, metrics beyond `Health()`, `OnPredicatePanic` hook) into TODO-able tasks — not started.
5. **Adopting go-etag's benchmark-baseline convention** (`reports/bench/<date>_<name>.txt`, `-benchmem -count=6`) — proposed only (f14).

## d) TOTALLY FUCKED UP

1. **Master CI red for 5 days and worsening.** Green 2026-09-10 → red 09-13 (Lint only;
   Test/Vet/Examples green — a lint-config break, consistent with the `exhaustruct`→`exhaustruct_v5`
   migration landing ungated) → still red 09-15 (Lint) → today Vet + Lint + Nix flake check all
   red. Severity: no merge-confidence signal for any consumer for 5 days; the last verified-green
   state is 2026-09-10. Nobody noticed for 5 days — including this session until it ran the
   report's mandated freshness checks.
2. **Ungated toolchain/dependency bump pushed by the auto-daemon (today's regression wave).**
   Commit `531cb96` (2026-09-18 19:04, message "chore: auto-commit 6 changed file(s) (heuristic)"):
   `go 1.26.7`→`1.27.1` directive, go-branded-id v0.5.1→v0.6.0, go-error-family v0.10.0→v0.10.1,
   `.golangci.yml` `exhaustruct`→`exhaustruct_v5` (+ `exclude:`→`ignore-patterns:` rename),
   flake.lock bump. Consequences: local build impossible (`GOTOOLCHAIN=local`, go1.26.7);
   CI Vet job fails (setup-go provides 1.26.x); Nix flake check fails (flake Go is 1.26 and the
   documented rule "vendorHash must be recomputed whenever go.mod/go.sum changes" was violated —
   no recompute happened). The changes sat UNCOMMITTED in the tree at session start (authored
   outside this session), and the daemon committed and pushed them mid-session without a gate.
   Process failure, not a code failure: manifests can enter the tree ungated and get pushed.
3. **`coverage-gate` exits 1 with a ZERO-byte log.** Reproduced twice with different, valid-looking
   cache setups (ambient `/mnt/buildcache` exists; explicit `/tmp/go-build-cache` +
   `/tmp/go-mod-cache`). The documented silent-exit gotcha was GOCACHE-only; today's variant
   hides even the underlying go error, which is presumably the toolchain mismatch (d1). A gate
   that fails silently and emptily is worse than no gate.
4. **CLTY is two months of propagated garbage.** Born garbled in the initial ROADMAP commit
   (`5f2ba13`), never decodable, yet copied through ≥9 harvest tables across 9 sessions — every
   docs-health pass included. Harvest hygiene has no "decode or flag" rule, so nonsense
   accumulated seniority by repetition.
5. **This session's own defects, bluntly.** (5a) Claimed mid-session "go-sse has no WHATWG §9.2
   conformance account" without grepping docs/ first — verification showed a full conformance
   program exists (archived); the real gap is only the missing living account (a5). Exactly the
   verify-before-claiming failure the global AGENTS.md warns about. (5b) First `go vet` run
   echoed "vet exit: 0" because `$?` captured `tail`'s exit, not vet's — the documented
   pipeline-masking trap; the visible toolchain error line prevented a wrong conclusion, and the
   first background coverage-gate run was likewise accepted at face value ("Exit code 1") before
   being properly re-captured. Caught both within the session; both should never have happened.

## e) WHAT WE SHOULD IMPROVE

| IMP  | Improvement (process, not product)                                                                                                                                                                                                                               | Priority |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- |
| IMP1 | Gate manifests: go.mod / go.sum / flake.lock / .golangci.yml must never reach master without verify.sh + flake check. Options: daemon exclusion list for manifests, or a required CI gate. The 09-13 Lint break AND today's wave both shipped through this hole. | high     |
| IMP2 | Make coverage-gate fail LOUDLY: propagate the underlying go failure to stderr, never exit with an empty log (extends the open TODO_LIST GOCACHE item to the zero-output variant seen today).                                                                     | high     |
| IMP3 | Harvest hygiene rule: any token that cannot be decoded gets flagged `⚠ unverified-origin` at harvest time, never silently copied forward (CLTY class).                                                                                                           | med      |
| IMP4 | Toolchain bumps are cross-cutting: go.mod directive + flake devShell Go + ci.yml go-version + golangci-lint pin (flake ↔ CI must stay aligned) + both vendorHashes. One checklist step; today's wave bumped manifests only.                                      | high     |
| IMP5 | Session discipline: grep before asserting a docs gap; capture true exit codes without filters (rule already exists — it was still tripped twice today, see d5).                                                                                                  | med      |
| IMP6 | Standing session-opener check: `gh run list --workflow ci.yml` before any work. Would have surfaced the 09-13 red five days earlier; today it was found only because the report format demanded gate state.                                                      | med      |

## f) Up to 50 things we should get done next

### P0 — unblock master (Critical)

| # | Item                                                                                                                                                          | Impact   | Effort | Category |
| - | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | -------- |
| 1 | Decide keep-vs-revert for the `531cb96` bump (gated on g1)                                                                                                    | Critical | S      | Decision |
| 2 | If REVERT: restore go 1.26.7 pins, `.golangci.yml` exhaustruct block, flake.lock; re-run `scripts/verify.sh --fast`; push                                     | Critical | M      | Bug      |
| 3 | If KEEP: bump flake devShell Go to 1.27.x + ci.yml go-version + golangci pin alignment + recompute vendorHash/vendorHashSsetest (lib.fakeHash → build → copy) | Critical | L      | Bug      |
| 4 | Fix the 09-13 Lint red on its own terms: reproduce golangci failure at `34749630537` (likely exhaustruct_v5 naming/version incompatibility), fix or revert    | Critical | S      | Bug      |
| 5 | Fix coverage-gate silent exit: print underlying go error, non-empty failure log (extends TODO_LIST's GOCACHE item)                                            | High     | S      | Tooling  |
| 6 | After gates green: push, verify CI green, then measure + record the coverage line (retire this report's N/A)                                                  | Critical | M      | Process  |

### P1 — session-derived doc/roadmap work (High)

| #  | Item                                                                                                                                                                        | Impact | Effort | Category |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | -------- |
| 7  | ROADMAP: advance sequencing — §1 exit criteria met; move Now→Developer experience; fold 3 leftover design questions into §5 with sources                                    | High   | S      | Docs     |
| 8  | ROADMAP §2: cross-link sentence — an in-tree Redis/PG EventStore re-opens the parked common/server split (§4)                                                               | High   | S      | Docs     |
| 9  | ROADMAP §3: delete or decode CLTY per g2; fallback wording "SSE extension fields (custom fields)"                                                                           | High   | S      | Docs     |
| 10 | ROADMAP §4 parked split: add go-etag cross-ref (deprecated.go shim + export-parity suite + tombstone doc.go = proven migration playbook)                                    | Med    | S      | Docs     |
| 11 | ROADMAP §5 typed-code raw idea: add go-etag `server/code.go` as reference implementation (declarative templates + bidirectional pinning test)                               | Med    | S      | Docs     |
| 12 | Promote a LIVING spec-conformance account: distill archived SUPERB plan D1–D6 + closeout into a docs/ doc (go-etag rfc9111-conformance.md analog), link from AGENTS gotchas | Med    | M      | Docs     |
| 13 | Re-verify wire-only consumer count (2→?) across sibling repos; update ROADMAP §4 trigger status (g3)                                                                        | Med    | M      | Docs     |
| 14 | Adopt benchmark-baseline convention from go-etag: `reports/bench/<date>_<name>.txt`, `-benchmem -count=6` before/after perf changes                                         | Med    | S      | Quality  |
| 15 | Repair truncated TODO_LIST.md cell (flake-update row ends mid-sentence: "error-swallowing `")                                                                               | Med    | S      | Docs     |
| 16 | If keeping the bump: read go-branded-id v0.6.0 + error-family v0.10.1 changelogs; consumer-side per go-ecosystem-upgrade sweep                                              | High   | M      | Quality  |

### P2 — existing TODO_LIST backlog (verified open 2026-09-03; re-verify at harvest)

| #  | Item                                                                                                                         | Impact | Effort | Category |
| -- | ---------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | -------- |
| 17 | `Stream.Send` partial-write semantics test (short-write fake)                                                                | High   | S      | Quality  |
| 18 | Context-cancellation mid-stream teardown test (clean close, no panic/deadlock)                                               | High   | M      | Quality  |
| 19 | `errors.AsType` migration evaluation sweep (Go 1.26+; never regress sentinel matching)                                       | Med    | M      | Quality  |
| 20 | Pin real `benchmem` numbers behind FEATURES' allocation-free hot-path claim                                                  | Med    | S      | Docs     |
| 21 | Confirm flake-update cron created its PR post-2026-09-07 (first run with PR permission)                                      | Med    | S      | CI       |
| 22 | Single-source golangci-lint version (ci.yml ↔ flake.nix enforcement, not comment)                                            | High   | S      | CI       |
| 23 | CI treefmt/format gate job (formatting currently local-only)                                                                 | Med    | S      | CI       |
| 24 | CI concurrency group to cancel superseded master runs                                                                        | Low    | S      | CI       |
| 25 | Dependabot/Renovate for Actions SHA pins (Node 20 deprecation warnings active)                                               | Med    | S      | CI       |
| 26 | Example smoke script: boot each example server, curl one event, kill                                                         | Med    | S      | Quality  |
| 27 | SECURITY.md with private vulnerability reporting contact                                                                     | Med    | S      | Docs     |
| 28 | CONTRIBUTING release-checklist additions (fuzz budget, govulncheck pin refresh, tag-signing policy, datastar pin-bump owner) | Med    | S      | Docs     |
| 29 | Godoc examples for `RequireDataJSON` and `WithOnDrop`                                                                        | Low    | S      | Docs     |
| 30 | `docs/guides/eventstore-patterns.md` — retention/GC for replay stores                                                        | Med    | M      | Docs     |
| 31 | `docs/guides/` filters and fan-out patterns (read-lock predicate contract)                                                   | Med    | M      | Docs     |
| 32 | Cross-repo CI: assert go-datastar tests against latest ssetest tag                                                           | Med    | S      | CI       |

### P3 — harvest candidates routed via docs-health (ROADMAP fuel; apply routing rigor)

| #  | Item                                                                                                               | Impact | Effort | Category   |
| -- | ------------------------------------------------------------------------------------------------------------------ | ------ | ------ | ---------- |
| 33 | Replay pagination (limit/cursor) API sketch for huge reconnect gaps                                                | Low    | M      | Feature    |
| 34 | `WithDrainPollInterval` + richer ShutdownResult — only on consumer-reported drain-visibility pain                  | Low    | S      | Feature    |
| 35 | Example servers as flake apps (`nix run .#datastar` etc.) — decide once, stop re-deferring                         | Low    | S      | Decision   |
| 36 | CSP headers + SRI posture for example vendored assets                                                              | Low    | S      | Security   |
| 37 | `docs/status/INDEX.md` — only if generated from git history, never hand-maintained                                 | Low    | M      | Docs       |
| 38 | `docs/guides/getting-started.md` distinct from README quickstart                                                   | Low    | M      | Docs       |
| 39 | Topic/channel multi-broadcaster routing — only on a real multi-hub consumer need                                   | Low    | L      | Feature    |
| 40 | samber/do lifecycle adapter revisit — only on concrete consumer ask                                                | Low    | S      | Feature    |
| 41 | Ecosystem dependency-policy doc for larsartmann/* (pin-vs-track; test coupling to upstream incidental behavior)    | Low    | M      | Docs       |
| 42 | Bound the `OnPredicatePanic` observability design question (currently silent recovery)                             | Low    | S      | Decision   |
| 43 | Bound the backpressure policy design question (block vs spill vs drop-on-full)                                     | Low    | M      | Decision   |
| 44 | Bound metrics-beyond-Health design (drop counters, per-subscriber stats)                                           | Low    | M      | Decision   |
| 45 | Client `Dial` helper — stays deferred until a concrete client consumer exists (do not pre-build)                   | Low    | —      | Deferred   |
| 46 | In-memory `EventStore` impl — only after item 8's split cross-link is resolved                                     | Low    | M      | Feature    |
| 47 | Drop `replace` directives from next datastartest tag (go-datastar checklist)                                       | Low    | S      | Cross-repo |
| 48 | CI headless browser test — stays BLOCKED on E2E scope decision (chromedp brainstorm Option B vs C)                 | Low    | L      | Blocked    |
| 49 | Misleading auto-commit `38e79aa` message — stays WONT (no history rewrite) unless ordered                          | Low    | —      | WONT       |
| 50 | Re-run docs-health AUDIT over docs/status after master unblocks (no reports exist for 09-04→09-18, the 5 red days) | Med    | M      | Docs       |

**HARVEST note:** f1–f16 and f50 are bounded and should land in TODO_LIST.md; f33–f49 are
ROADMAP fuel. Nothing is harvested yet — explicit wait order in effect.

## g) Questions I CANNOT figure out myself

1. **Keep or revert the `531cb96` bump?** go directive 1.27.1 + go-branded-id v0.6.0 +
   go-error-family v0.10.1 + exhaustruct_v5 + flake.lock, authored outside this session, committed
   and pushed ungated by the daemon, now breaking Vet/Lint/Nix-flake-check on master. I tried:
   diff forensics, CI job histories for all three red pushes, local repro — the CAUSE is certain;
   the INTENT is not. Reverting risks destroying a bump you wanted; keeping it commits me to a
   toolchain rollout (flake Go, ci.yml go-version, golangci pin, both vendorHashes) I should not
   start unilaterally.
2. **What did "CLTY" originally mean?** Born garbled in the initial ROADMAP commit (`5f2ba13`),
   unreadable at birth, propagated through ≥9 harvest cycles. Grepped the repo and full git
   history — no decode exists anywhere. If you don't know either, item f9 deletes it.
3. **Which repos consume go-sse wire-only today?** The parked split's re-open trigger is
   "2 → 3 wire-only consumers", and the 2-of-4 count dates to 2026-07-25. The consumer repos'
   current go.mod pins are the ground truth; they are not all locally greppable and a cross-repo
   sweep is your call, not mine.
