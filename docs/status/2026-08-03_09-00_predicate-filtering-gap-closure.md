# Status: Predicate Filtering — Documentation & Test Gap Closure

**Date:** 2026-08-03 09:00
**Session scope:** Close all documentation and test gaps identified in the predicate-filtering self-review (`2026-08-03_07-03_predicate-filtering-self-review.md`)
**Verdict:** Shipped with minor debt remaining

---

## a) FULLY DONE

### Documentation updates (4 files)

| File | What was added |
|---|---|
| `broadcaster.go` | New `# Filtered Subscriptions` section in the `Broadcaster` doc comment: predicate contract (pure, fast, non-blocking), panic semantics (crashes by design, no recover), equivalence (`Subscribe()` = `SubscribeFilter(nil)`), usage example |
| `FEATURES.md` | `SubscribeFilter` row in fan-out table (FULLY_FUNCTIONAL); `FilteredEventStore` + `ReplayFiltered` rows in replay table; updated integration test, race test, example test, and benchmark rows to include the new tests |
| `README.md` | `SubscribeFilter` + `FilteredEventStore` in "What's Included" table; new "Filtered subscriptions" section with code examples; `SubscribeFilter` in Broadcaster API reference; `FilteredEventStore` interface + `ReplayFiltered` in Replay API reference; new design decision entry ("Predicate under read lock") |
| `docs/DOMAIN_LANGUAGE.md` | 4 new glossary terms: Predicate, SubscribeFilter, FilteredEventStore, ReplayFiltered |

### Test improvements (3 added, 1 rewritten)

| Test | What it verifies |
|---|---|
| `TestSubscribeFilter_BroadcastManyRespectsPredicates` | `BroadcastMany` honors per-subscriber predicates; filtered subscriber gets only matching events in order while unfiltered gets all |
| `TestSubscribeFilter_ConcurrentRace` (rewritten) | Correctness under concurrency: 4×2000 concurrent broadcasts (mixed match/skip) + 500 subscribe/unsubscribe churn; persistent filtered subscriber drained into atomic counters; asserts zero non-matching events delivered |
| `TestIntegration_SubscribeFilter` | HTTP round-trip: non-matching broadcasts never reach the client over the wire; only the matching event appears in the SSE body |
| `BenchmarkSubscribeFilter_PredicateOverhead` | Unfiltered (nil pred) vs filtered (always-true pred) at 1/100/1000 subscribers to isolate per-subscriber predicate call cost |

### CHANGELOG

- Added entries for `BenchmarkSubscribeFilter_PredicateOverhead`, `TestSubscribeFilter_BroadcastManyRespectsPredicates`, `TestIntegration_SubscribeFilter` under `[Unreleased] > Added`

### Verification (all green)

- `go test ./... -race -count=1` — all pass
- `golangci-lint run ./...` — 0 issues
- `nix fmt` — 9 files formatted, 0 changed
- `nix flake check` — all checks passed

---

## b) PARTIALLY DONE

### DOMAIN_LANGUAGE.md line references — partially correct

| Term | Referenced line | Actual line | Status |
|---|---|---|---|
| `subscriber[T].pred` | `fanout.go:72` | `fanout.go:75` (type), `:77` (field) | Off by 3 (referenced doc comment, not declaration) |
| `SubscribeFilter` | `fanout.go:142` | `fanout.go:142` | Correct |
| `FilteredEventStore` | `replay.go:25` | `replay.go:25` | Correct |
| `ReplayFiltered` | `replay.go:70` | `replay.go:70` | Correct |

### Race test correctness — partially verified

The rewritten `TestSubscribeFilter_ConcurrentRace` verifies that **zero non-matching events** were delivered to the persistent filtered subscriber. It does NOT verify:
- That a reasonable **count** of matching events was delivered (only checks `> 0`, not `>= some threshold`)
- That the subscribe/unsubscribe churn subscribers also received only matching events (they're created and immediately destroyed — no collector)
- No data race between `Shutdown`/`drain` and filtered broadcast paths

---

## c) NOT STARTED

1. **ReplayFiltered HTTP integration test** — The replay filter tests (`replay_filter_test.go`) use `httptest.ResponseRecorder`, not a real HTTP server. No test exercises `ReplayFiltered` over an actual `httptest.Server` round-trip the way `TestIntegration_LastEventIDReconnectionReplay` does for plain `Replay`.
2. **Shutdown + filtered subscribers test** — No test verifies that `Shutdown(ctx)` drain works correctly when some subscribers have predicates. The drain path checks `len(sub.ch)` which should be unaffected by predicates, but there's no explicit coverage.
3. **Predicate panic safety documentation in ReplayFiltered** — `SubscribeFilter` documents the "panics crash by design" contract in `broadcaster.go`. `ReplayFiltered` has the same risk (fallback path calls `pred(evt)` in a loop) but the panic contract is not documented on `ReplayFiltered` itself.
4. **v0.4.0 release** — CHANGELOG is under `[Unreleased]`, no git tag, no release notes. Requires explicit user approval (irreversible).
5. **cqrs-htmx `JournalSSEStore` → `FilteredEventStore`** — Separate project. The efficient path exists in go-sse but no consumer implements it yet.
6. **Push to origin** — 4 commits ahead of `origin/master`. Not pushing without explicit instruction.

---

## d) TOTALLY FUCKED UP

### Nothing catastrophic, but two honest misses:

1. **Stale line references in DOMAIN_LANGUAGE.md that I walked past.** The pre-existing entries for `Subscribe` (`fanout.go:39`, actually 127) and `Broadcast` (`fanout.go:89`, actually 196) were already wrong before my session. I read the file, added correct references for my new terms, and **left the stale ones untouched**. I should have fixed them while I was there — the 15-second fix was staring me in the face.

2. **Pre-existing `_ = i` code smell in `TestSubscribeFilter_BufferOnlyFillsWithMatching`** (`filter_test.go:123-125`). Uses `for i := range 10 { ...; _ = i }` instead of `for range 10`. I read this exact file, added tests to it, and didn't fix the smell. Classic "fix issues on sight" failure.

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements

1. **Line references are a maintenance debt trap.** DOMAIN_LANGUAGE.md hardcodes `file.go:NN` references that rot on every edit. Options: (a) drop line numbers entirely and reference symbol names only, (b) generate them with a script, (c) accept staleness and audit periodically. Current state: half-stale, half-correct, which is the worst of all worlds.

2. **The race test rewrite, while better, still doesn't verify throughput.** Checking `received.Load() > 0` is barely better than `== 0`. With 4×1000 matching events, we should verify at least, say, 500 were received (accounting for non-blocking drops during churn). The threshold doesn't need to be exact, just non-trivial.

3. **ReplayFiltered has no HTTP integration test.** Every other major path (direct send, broadcaster fan-out, heartbeat, reconnection replay, DataStar wire format, and now SubscribeFilter) has an HTTP round-trip test. ReplayFiltered is the only public API without one. This is an asymmetry.

4. **Predicate panic contract is documented inconsistently.** `broadcaster.go` has it; `ReplayFiltered` in `replay.go` doesn't. Same risk, different documentation depth. Should be consistent.

5. **The benchmark ran with `-benchtime=10x`** (10 iterations). The results showed the filtered path being *faster* than unfiltered at 1 subscriber (237 ns vs 280 ns) which is almost certainly measurement noise, not a real signal. I didn't flag this or run a proper benchmark. The benchmark code is correct; the *verification* was insufficient.

### Code quality observations (pre-existing, not introduced this session)

6. **`TestSubscribeFilter_BufferOnlyFillsWithMatching`** has a dead `_ = i` variable and could use `for range 10` instead of `for i := range 10 { _ = i }`.

7. **`DOMAIN_LANGUAGE.md` line references for `Subscribe` and `Broadcast`** are stale from a prior refactor that moved them. These were wrong before my session and I left them.

---

## f) Up to 50 things to get done next

### High priority (correctness gaps)

1. Fix stale line references in DOMAIN_LANGUAGE.md (`Subscribe` → `fanout.go:127`, `Broadcast` → `fanout.go:196`)
2. Fix `subscriber[T].pred` reference (`fanout.go:72` → `fanout.go:75`)
3. Fix `_ = i` smell in `TestSubscribeFilter_BufferOnlyFillsWithMatching`
4. Add `TestIntegration_ReplayFiltered` — HTTP round-trip with `FilteredEventStore` and fallback store
5. Add predicate panic contract documentation to `ReplayFiltered` doc comment
6. Add `TestSubscribeFilter_ShutdownDrainsFilteredSubscribers` — verify Shutdown works with predicate subscribers

### Medium priority (test quality)

7. Strengthen race test: assert `received.Load() >= 500` (not just `> 0`)
8. Add race test subscriber churn with collector goroutines (verify churn subscribers also get only matching events)
9. Add `TestReplayFiltered_FallbackPredicatePanic` — document/test that a panicking predicate in fallback crashes (consistency with SubscribeFilter contract)
10. Run `BenchmarkSubscribeFilter_PredicateOverhead` with proper `-benchtime=3s` and record results in FEATURES.md or a benchmark doc
11. Add `TestSubscribeFilter_BroadcastManyMixedSubscribers` — half filtered, half unfiltered, verify correct partition
12. Add `TestSubscribeFilter_DropPolicyRespected` — filtered subscriber with full buffer still drops matching events (not non-matching)

### Documentation

13. Add `SubscribeFilter` to doc.go "Concurrency and Memory Model" section
14. Add `ReplayFiltered` to doc.go "Reconnection" section (currently only mentions `Replay`)
15. Update `ROADMAP.md` — mark the "topic routing" raw idea as resolved by predicate filtering (may already be done from prior session; verify)
16. Consider a "Predicate design guide" doc: when to use `SubscribeFilter` vs multiple broadcasters vs topic-based fanOut
17. Add `SubscribeFilter` to the `example/` server package as a usage reference

### Release

18. Get user approval for v0.4.0 cut
19. Move `[Unreleased]` → `[0.4.0]` in CHANGELOG with date
20. Add release notes (GitHub release body)
21. Tag `v0.4.0`
22. Push to origin

### Consumer integration (separate projects)

23. DiscordSync: decide keep-or-revert the cqrs-htmx→go-sse alias migration commits
24. DiscordSync: test the migrated code if keeping the migration
25. cqrs-htmx: extend `JournalSSEStore` to implement `FilteredEventStore` (push predicate into journal query)
26. cqrs-htmx: re-export `SubscribeFilter` / `ReplayFiltered` through existing alias layer (or document direct go-sse import)

### Architecture / future

27. Consider a `FilteredBroadcaster[T]` convenience type that wraps a predicate at construction (for single-filter broadcasters)
28. Consider `WithFilter[T](pred)` option for `NewBroadcaster` (applies predicate to all subscribers)
29. Evaluate whether `FilteredEventStore` should take a context parameter for cancellation
30. Consider metrics hooks for filtered drops (predicate rejected vs buffer full — currently indistinguishable)
31. Consider `BroadcastFiltered(pred, msg)` — broadcast only to subscribers whose predicates match the predicate (meta-filtering)
32. Add a fuzz test for the fallback `ReplayFiltered` post-filter loop (edge cases in predicate return values)

### Cleanup / maintenance

33. Audit all `file.go:NN` references across docs for staleness
34. Consider replacing line-number references with symbol-only references (maintenance-free)
35. Run `govulncheck` after dependency changes
36. Verify CI pipeline passes on the new test additions (`.github/workflows/ci.yml`)
37. Check if `gofumpt`/`golines` line-length limits affect the new doc comments
38. Add `SubscribeFilter` to the `example/datastar` example if relevant

### Verification gaps

39. No test for `SubscribeFilter` with `WithBufferSize(1)` — smallest possible buffer + predicate
40. No test for `Unsubscribe` of a filtered channel mid-broadcast (channel closed while predicate running)
41. No test for `Close()` while a predicate is executing (write lock vs read lock interaction)
42. No test for `OnSubscribe`/`OnUnsubscribe` hooks firing correctly with `SubscribeFilter`/`Unsubscribe`
43. No test for `SubscriberCount()` including filtered subscribers (should — they're in the same map)
44. No test for `Health()` reporting filtered subscribers in `SubscriberCount`
45. No test for concurrent `SubscribeFilter` + `SubscribeFilter` (two filtered subs, different predicates)
46. No test for a predicate that captures and mutates external state (documenting the "pure" contract by negative example)
47. No test for `BroadcastMany` with 0 messages + filtered subscriber
48. No test for `ReplayFiltered` with empty store + predicate
49. No test for `ReplayFiltered` where all events are filtered out (returns 0, no error)
50. No test for `ReplayFiltered` efficient path where `EventsAfterFiltered` returns error

---

## g) Questions I cannot answer myself

### 1. DiscordSync migration: keep or revert?

DiscordSync repo has unverified auto-committed commits (`b725ffed`, `5b1a9f6e`) that migrated from `cqrs-htmx` SSE aliases to direct `go-sse` imports. These were never tested. I cannot determine whether this migration is correct without reading DiscordSync's full SSE usage and its `cqrs-htmx` dependency surface — which is a separate project with its own AGENTS.md, test suite, and architecture. Should I:
- **(a)** Leave it — you'll handle DiscordSync separately
- **(b)** Investigate and verify/test the migration now
- **(c)** Revert the commits and redo the migration properly in a later session

### 2. Cut v0.4.0 now or after cqrs-htmx integration?

The predicate filtering API is shipped, tested, and documented. But no consumer implements `FilteredEventStore` yet — the efficient replay path is theoretical. Cutting v0.4.0 now locks the API surface before it's been validated by a real `FilteredEventStore` implementation. Alternatively, waiting until cqrs-htmx's `JournalSSEStore` implements it would validate the interface design. This is a product/release-timing decision I cannot make for you.

### 3. Predicate panic policy: confirm "let it crash"?

I documented the contract as "a panicking predicate will crash the broadcaster goroutine (do not recover; fix the predicate)" — consistent with Go's philosophy and the existing `MustParseEventID` pattern. But there's an alternative: wrap the predicate call in a `defer/recover` that logs and removes the offending subscriber. This would make one bad predicate non-fatal to other subscribers. I chose "let it crash" because it's simpler and honest, but this is a philosophy decision that affects the library's fault-tolerance contract. Is "let it crash" the right call, or do you want a `SubscribeFilterSafe` variant with recover?

> **Update 2026-08-03 (commit `b666ed5`, v0.4.0):** **REVERSED — panics are now recovered.** The user's directive was "we should NEVER panic!" `safePredCall[T]` in `fanout.go` wraps every predicate call in `defer/recover`; a panicking predicate returns `false` (treated as non-match) and the event is skipped for that subscriber. This applies to both `SubscribeFilter`'s fan-out path and `ReplayFiltered`'s fallback path. No `SubscribeFilterSafe` variant was needed — recovery is now the default and only behavior. The "let it crash" contract documented above is **false as of v0.4.0**; the README, AGENTS.md, doc.go, broadcaster.go, and replay.go all document the recovery contract.

---

## Session metrics

| Metric | Value |
|---|---|
| Files modified | 6 (`broadcaster.go`, `FEATURES.md`, `README.md`, `docs/DOMAIN_LANGUAGE.md`, `CHANGELOG.md`, `filter_test.go`, `integration_test.go`) |
| Tests added | 2 (`BroadcastManyRespectsPredicates`, `Integration_SubscribeFilter`) |
| Tests rewritten | 1 (`ConcurrentRace` — now verifies correctness, not just no-panic) |
| Benchmarks added | 1 (`PredicateOverhead`) |
| Doc sections added | 5 (broadcaster.go, README "Filtered subscriptions", README design decision, DOMAIN_LANGUAGE glossary, doc.go already done prior) |
| Commits this session | 4 (auto-committed by daemon) |
| Commits ahead of origin | 4 |
| Verification | `go test -race` pass, `golangci-lint` 0 issues, `nix flake check` pass |

---

## Resolution (2026-08-03)

Open items from section c and section f are tracked in TODO_LIST.md.

| Item | Resolution |
|------|------------|
| §d.1 Stale DOMAIN_LANGUAGE line refs (Subscribe, Broadcast) | FIXED — all line refs replaced with symbol-only references (maintenance-free) |
| §d.2 `_ = i` code smell in `filter_test.go` | FIXED — changed to `for range 10` |
| §c.1 ReplayFiltered HTTP integration test | Still open — TODO_LIST "Predicate filtering correctness gaps" |
| §c.2 Shutdown + filtered subscribers test | Still open — TODO_LIST |
| §c.3 Panic contract doc on ReplayFiltered | Still open — TODO_LIST |
| §c.4 v0.4.0 release | Still open — TODO_LIST "Release" |
| §c.5 cqrs-htmx JournalSSEStore FilteredEventStore | Cross-project — ROADMAP §2 (developer experience) |
| §c.6 Push to origin | User decision — 5+ commits ahead of origin/master |
| Q1: DiscordSync migration | Still open — cross-project decision |
| Q2: Cut v0.4.0 now or after cqrs-htmx? | Still open — TODO_LIST |
| Q3: Predicate panic policy | "Let it crash" adopted; ReplayFiltered doc still needed (TODO_LIST) |
