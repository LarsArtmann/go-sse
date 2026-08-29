# Status: SSE Spec Conformance Execution — go-sse + datastartest

**Date:** 2026-08-16 10:14
**Session:** Executing `docs/planning/archived/2026-08-16_09-21_SUPERB-spec-based-hardcore-sse-conformance-testing.md` (tasks C1–C12 / F1–F22)
**Scope:** Everything below reflects only this session's run and what was directly observed in it.

---

## a) FULLY DONE (verified green at time of completion)

### Parser conformance rewrite — `ssetest/reader.go` (F1–F6) ✅

- New `splitSSELines` bufio.SplitFunc: CR, LF, CRLF all terminate lines (§9.2.5); trailing buffered CR held back so CRLF is always ONE terminator, incl. CRLF split across reads and lone CR at EOF.
- `bomStripReader`: strips exactly one leading UTF-8 BOM; probes once via `io.ReadFull`, replays short reads unchanged, defers probe EOF until replayed bytes drain.
- `streamParser` type: per-frame state (Type, DataLines) vs connection-level sticky state (`lastID`, `retry`) — modeled on the spec's dispatch algorithm.
  - `id:` updates sticky buffer; NUL-containing id ignored (D4); empty `id:` resets to `""`; dispatched events snapshot the buffer (D5 fixed).
  - `retry:` sticky too; invalid values never reset prior value; `ParseUint(value, 10, 64)`.
  - EOF discards pending frame (D2 fixed) — post-loop `dispatchFrame` removed from both `ReadEvents` and `ReadNEvents`.
- `Event` doc comments updated to the sticky semantics.

### WPT corpus transcription (F7–F10) ✅

- `ssetest/wpt_format_corpus_test.go`: 17 WPT `eventsource/format-*` vectors, 4 spec §9.2.6 example streams, 8 Chromium `event_source_parser_test.cc` cases — every case carries its upstream citation. All green.
- Includes derived-but-documented vectors (format-field-retry-empty, TrailingCREndsLineAtEOF) with derivation notes.

### Test-suite corrections (F11) ✅

- `TestReadEvents_NoTrailingBlankLine` flipped → `TestReadEvents_IncompleteFinalFrameDiscarded` with spec citation.
- `partialReader` fixture rewritten to serve bytes across reads (the BOM probe consumes the first 3 bytes through its own Read; one-shot fixtures starve the scanner).

### Fuzz hardening (F12) ✅

- `FuzzReadEvents`: 25 new conformance seeds (BOM, BOM-fragments, NUL id, sticky id, lone CR states, EOF-discard shapes) + byte-by-byte chunk-invariance property (parse(wire) == parse(one byte at a time)). ~1.55M execs, 0 failures.

### Writer goldens + ParseEventID (F13, F14) ✅

- `event_spec_test.go` (root): stock-ticker vector, space-after-colon note, field order, empty-data dispatch, CR/CRLF splitting table, heartbeat comment form, `ParseEventID` value-space + actionable-error tests.
- `event.go`: `ParseEventID` now rejects NUL (`ContainsAny(s, "\n\r\x00")`), message and docs updated (D6 fixed).

### Chunk-boundary independence (F15) ✅

- `chunkedReader` helper + `chunk_boundary_test.go`: entire corpus re-parsed at chunk sizes 1/2/3/5/7/4096 vs baseline; plus targeted CRLF-split-across-reads, lone-CR, and `\r\r` subtests. All green.

### Round-trip closure (F16–F17) ✅

- `roundtrip_test.go`: table (9 cases) + heartbeat/retry invisibility + `FuzzWriteReadRoundTrip` (identity modulo CR/CRLF→LF and structural trailing LF). ~444k execs, 0 failures.
- **Real discovery:** trailing LF in `Event.Data` is structurally unrepresentable — `splitLines("x\n")` yields `["x"]` and the spec strips one trailing LF from the data buffer at dispatch. Pinned in table test, fuzz property (`TrimSuffix`), and a committed regression seed (`ssetest/testdata/fuzz/FuzzWriteReadRoundTrip/2ba7b6a0aaf94e65`).

### Docs (F18, mostly) ✅

- `ssetest/README.md`: new "Spec conformance" section.
- `CHANGELOG.md`: full Unreleased entry (Added/Fixed/Changed) with per-deviation detail.
- Root `AGENTS.md`: replaced obsolete `bufio.ScanLines` gotcha with 4 new gotchas (conformance contract, splitSSELines, sticky id/retry, trailing-LF structural rule); updated `ParseEventID` note and module table.

### datastartest sync (F19) ✅

- go-datastar repo locally present — full port applied: `reader.go` (same parser), `collect.go` `ReadNEvents` loop, flipped EOF test, `wpt_format_corpus_test.go` (17 WPT + 3 spec + 8 Chromium), `chunk_boundary_test.go` + helper.
- Tested inside its Nix devShell (repo needs Go 1.26.6; shell Go is 1.26.5, `GOTOOLCHAIN=local`): build, vet, full suite, and 5s fuzz smoke — all green.
- go-datastar `CHANGELOG.md` updated; go-sse AGENTS.md sync-note updated to "applied to both".

### Docs inventory (F20) ✅/partial

- `FEATURES.md`: ssetest section rewritten (5 new rows: conformant parser, WPT corpus, chunk independence, round-trip, updated dataless-frame evidence).

---

## b) PARTIALLY DONE

### F21 — Full Nix verification ⚠️ (mid-flight, interrupted by this report)

- `nix fmt` — ran once (4 files reformatted); **not re-run after subsequent manual edits**.
- `nix run .#test-race` — GREEN (root, ssetest, datastar example) — but **predates the final reader.go lint edits** (nolint comments, `crlfLen` const, unnamed returns, `range data` loop, `Read(buf)` rename). Behavior-neutral on paper; not re-verified.
- `nix run .#vet` — green earlier; not re-run after final edits.
- `nix run .#lint` — **NOT GREEN. 18 issues found; I fixed reader.go's batch (via `//nolint` where justified, real fixes otherwise) and event_spec_test.go's 2 (nlreturn, varnamelen). Remaining ~8 are in test files I wrote:**
  - `dupword` ×3 (`reader_fuzz_test.go:34`, `wpt_format_corpus_test.go:164,175` — WPT names/comments contain repeated words like "test test"/"data data")
  - `varnamelen` ×5 (`tc` in chunk_boundary/roundtrip/wpt corpus loops, `p` in chunkedReader.Read)
- `nix flake check` — **NOT RUN AT ALL this session.**
- `nix run .#coverage` — not run; FEATURES.md still cites the stale 94.6% figure.

### F18 remainder

- datastartest `README.md` has no conformance section (go-sse's does). Not ported.

### F19 remainder

- datastartest `FuzzReadEvents` seeds were NOT extended with the WPT vectors (ssetest's were). Chunk corpus + parser are synced; fuzz seed list is not.

---

## c) NOT STARTED

- **F22 — commits + push.** Zero commits of any implementation work (working tree has 9 modified + 6 new paths in go-sse, uncommitted; the auto-git daemon has not picked them up either). Nothing pushed.
- Plan file status line still says `Status: EXECUTING` — not updated with completion state.
- go-datastar side: its changes are also uncommitted (see d/e for the dirty-tree complication).
- `TODO_LIST.md` untouched (currently "No open items" — nothing added for follow-ups found this session).

---

## d) TOTALLY FUCKED UP (caught & fixed in-session; listed so they're on record)

1. **`splitSSELines` index-out-of-range (first draft):** `data[i+1]` evaluated when `i` is the last byte. Caught by re-reading before testing; restructured with `i+1 < len(data)` guard.
2. **Hand-rolled `io_EOF` sentinel** in the first `chunkedReader` draft — `bufio` compares against `io.EOF` by identity, so this would have silently broken every chunked test. Rewrote with real `io.EOF` immediately.
3. **Dead fuzz invariant:** the sticky-ID check never updated `lastID`, so it could only ever compare `"" == evt.ID` — and it coexisted with a non-compilable `events[i] != chunked[i]` struct comparison (slice fields). Both removed; replaced with the chunk-invariance property.
4. **Stray `requireEventsMatch` call** in the round-trip table would have failed every case with empty expectations. Caught on re-read before running.
5. **Wrong round-trip premise:** I asserted empty `Data` → zero events. Reality: `splitLines("")` returns `[""]`, the writer emits `"data: \n"`, and the spec-correct reader dispatches ONE empty event (exactly WPT format-field-data). Fixed the test; the behavior is correct and now pinned.
6. **Fuzz-found (genuine, kept):** trailing LF in `Data` doesn't survive the wire (see a/F16–F17). Not a bug — a spec-structural fact I had wrong; now pinned three ways.
7. **Corpus wire-construction bug:** format-field-data was missing the blank line between frames 2 and 3 (got 2 events, want 3). The corpus test caught it — the suite working as intended.
8. **Wrong wire in a chunk subtest:** used `"data: x\r\ndata: y"` where a lone-CR case was intended. Caught during writing.
9. **`partialReader` fixture breakage:** old one-shot Read fixture starved the BOM probe after the rewrite — test failure revealed the fixture's hidden assumption. Rewritten to serve across reads (lesson: reader fixtures must behave like streams).
10. **Comment-association wart (STILL PRESENT):** while lint-fixing, I inserted `const crlfLen` between the `splitSSELines` doc comment and the function — the doc block now documents the const, and `splitSSELines`'s doc is dangling above it. Cosmetic but sloppy; needs a 2-minute fix.

---

## e) WHAT WE SHOULD IMPROVE (process observations from this run)

1. **Run `.#lint` per-file as I write, not as a final battery.** 18 issues at once at the end; half were mechanical (varnamelen/dupword) that cost nothing early.
2. **Never draft a custom EOF/error sentinel when the stdlib identity-compares.** Cost a full file rewrite.
3. **Invariants in fuzz tests must be executable statements, not aspirations** — the dead sticky-ID check looked like coverage but asserted nothing.
4. **Fixture realism:** any reader handed to the new parser must serve across multiple `Read` calls (BOM probe + split-func holdback both read ahead). Document in AGENTS.md gotcha (done).
5. **Test premises deserve the same scrutiny as implementation** — failure #5 was me encoding a wrong mental model into a test, then being surprised the code was right.
6. **Finish the verification battery before writing status/asking to commit** — the lint-fix edits after the last green test-race left the tree in "probably fine, unverified" state.
7. **Keep doc comments attached when inserting consts mid-file** (wart #10).

---

## f) NEXT — up to 50 things, roughly priority-ordered

**Finish F21 (verification):**

1. Fix `dupword` ×3 in reader_fuzz_test.go + wpt_format_corpus_test.go (reword comments or `//nolint:dupword` with reason — WPT file names legitimately repeat words).
2. Fix `varnamelen` ×5: rename `tc` → `case`/`vector` (chunk_boundary, roundtrip, wpt corpus loops) and `p` → `buf` (chunkedReader.Read; reader.go Read already renamed).
3. Fix the `crlfLen`/doc-comment association wart in reader.go.
4. `nix fmt` (treefmt: gofumpt/golines will also settle formatting after manual edits).
5. `nix run .#test-race` — re-verify after lint edits.
6. `nix run .#vet`.
7. `nix run .#lint` → must be 0 issues.
8. 15s re-fuzz `FuzzReadEvents` + `FuzzWriteReadRoundTrip` smoke after final edits.
9. `nix run .#coverage` — refresh the ssetest coverage number.
10. `nix flake check` — the full hermetic gate, never run this session.
11. Update FEATURES.md 94.6% figure with the fresh coverage number.

**Finish F22 (ship go-sse):**
12. Stage & commit in logical groups per plan: (a) parser conformance rewrite + flipped tests; (b) WPT corpus + chunk-boundary tests; (c) round-trip + fuzz seeds + testdata regression seed; (d) writer goldens + ParseEventID NUL; (e) docs (CHANGELOG/AGENTS/README/FEATURES) + plan-file status.
13. Push go-sse master.

**Plan & docs closure:**
14. Update plan file `Status: EXECUTING` → `DONE` (or per-task checkmarks) + verification results section.
15. Add `docs/status/` link or note in plan file pointing at this report.
16. TODO_LIST.md: add the carry-over items from this report (datastartest fuzz seeds, coverage refresh cadence, etc.) rather than leaving "No open items".

**datastartest (go-datastar repo) closure:**
17. Extend datastartest `FuzzReadEvents` seeds with the WPT vectors (parity with ssetest).
18. Add conformance section to datastartest `README.md`.
19. gofmt-check the files I touched there via its devShell one more time (clean earlier; `event_fuzz_test.go` has a PRE-EXISTING gofmt issue — not mine, leave).
20. Commit ONLY my datastartest files (reader.go, collect.go, reader_test.go, 3 new test files, CHANGELOG.md) — the repo has unrelated pre-existing dirty files (ci.yml, dprint.json deletion, CONTRIBUTING.md, e2e_test.go) that must NOT be swept into my commit.
21. Push go-datastar (after user decision — see questions).
22. Note the sync in go-datastar's AGENTS.md (it documents the datastartest module table).

**Harden further (optional, high-value):**
23. Fuzz `splitSSELines` directly (byte-stream SplitFunc fuzz, comparing whole-stream vs chunked token sequences).
24. Property: `splitLines` (writer) and `splitSSELines` (reader) agree on line boundaries — cross-layer terminator equivalence.
25. Round-trip via HTTP once (existing e2e already covers; add a sticky-id assertion to an e2e reconnect test).
26. Add `Last-Event-ID` header → `ParseEventID` → id field end-to-end test in examples (reconnect path).
27. Port the roundtrip test concept to datastartest (go-datastar patch → `Event()` → wire → datastartest.ReadEvents → typed accessors identity).
28. Consider WPT `eventsource/*` non-format vectors worth transcribing (e.g., `EventSource` construction/CORS ones are browser-only — document why we skip).
29. Add a tiny "conformance" section to root README.md pointing at the corpus.
30. Root `doc.go`: one-line statement that writer output is pinned to §9.2.6 goldens.

**Housekeeping:**
31. Ensure `ssetest/testdata/fuzz/...` regression seed is committed intentionally (it is a crash-regression seed from the trailing-LF find).
32. Check `.gitignore` interactions for `testdata/` (currently not ignored — correct).
33. Re-check gopls diagnostics on files I touched after final edits (pre-existing stream.go:131 jsonv2 warning is unrelated — leave).
34. Consider `nix run .#coverage` gate still passing at ≥90% for root (library unchanged except ParseEventID, but verify).

_(34 concrete items; the remaining headroom to 50 would be speculative — stopping at what's real.)_

---

## g) QUESTIONS (cannot be answered from the repos/plan alone)

1. **go-datastar commit policy:** its working tree has unrelated pre-existing modifications (`.github/workflows/ci.yml`, deleted `dprint.json`, `CONTRIBUTING.md`, `datastartest/e2e_test.go`) that I did not author. May I commit and push ONLY my datastartest-sync files there, or do you want to handle that repo's dirty tree yourself first?
2. **Release posture for go-sse:** the Unreleased section now contains the sticky-ID/retry semantic change (breaking-ish for ssetest consumers). After push — cut `v0.6.0` (semver: behavior change), `v0.5.1` (fix-only framing), or leave Unreleased for now?
3. **Push authorization for go-sse itself:** the plan's F22 says "detailed commits + push", and I want to confirm that still stands post-execution (i.e., push to `origin/master` directly, no PR) given the branch tracks origin and is otherwise clean.

---

## Verdict

Core engineering is DONE and was green when last fully tested (parser fixes, full corpus, chunk independence, round-trip, both repos synced). What stands between here and "shipped": ~8 mechanical lint fixes in test files, the never-yet-run `nix flake check`, a post-edit re-verification pass, and the entire commit/push step. Nothing found suggests any of the six deviations were misdiagnosed — the corpus validated every fix it forced.

---

## Resolution (2026-08-29, docs-health pass)

- **F21/F22 finished by the closeout session** (`7776bc7` lint stability, `a5ff824` vendorHash split, `37e9791` plan closure) — see `2026-08-16_11-58_conformance-closeout-self-review.md`. All §b/§c items in this report are closed by that session.
- **Shipped:** `d6bea20` (parser conformance rewrite + corpus + writer goldens + ParseEventID NUL). Coverage claims refreshed: ssetest 95.3% (FEATURES.md, docs-health pass 2026-08-29).
- **§f verdicts (34 items):** 1–11 done at `7776bc7`, `a5ff824`, `37e9791` (lint fixes, crlfLen wart, fmt, test-race, vet, lint 0, fuzz smoke, coverage, flake check, FEATURES 94.6%→95.5%). 12–13 done at `d6bea20`+`37e9791` (logical commits, push). 14–16 done at `37e9791` (plan DONE + status link; TODO_LIST carry-over added). 17–22, 27 → go-datastar repo scope (17 fuzz seeds, 18 README conformance, 20 commit/push own files, 22 sync note done at go-datastar `83d7c60`/`496a18b` lineage; 19 gofmt leave; 21 push was user-approved). 23–26, 28, 30 → open, tracked in TODO_LIST.md (Parser parity & fuzz depth / Docs). 29 done (docs-health pass 2026-08-29 — README spec-conformance section). 31 done at `d6bea20` (regression seed committed). 32 done (verified: `testdata/` not gitignored). 33 known gopls friction → TODO_LIST (gopls hygiene). 34 done (`nix run .#coverage` gate green, ≥90%).
- **§g answers realized:** Q1 → cut v0.5.1 framing (done `ec43574`); Q2 daemon-guard → TODO_LIST (CI & tooling); Q3 push → done (`37e9791` lineage pushed).
