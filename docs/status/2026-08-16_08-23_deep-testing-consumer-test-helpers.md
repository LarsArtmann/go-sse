# Status Report: Deep Testing for Consumers (datastartest + ssetest)

_Date: 2026-08-16 08:23 (CEST)_
_Scope: `/home/lars/projects/go-datastar` (datastartest module) and `/home/lars/projects/go-sse` (new ssetest module). Triggered by: "my projects should make deep testing easy for all its consumers."_

---

## 1. WHAT IS FULLY DONE AND WORKING

### go-datastar — datastartest hardening (committed by auto-git daemon as `06bb019` + `b0482db`)

| Change | Detail | Verified by |
| --- | --- | --- |
| `testing.TB` everywhere | All 8 public helpers switched from `*testing.T` → `testing.TB` (works with `*testing.T`, `*testing.B`, Ginkgo's `GinkgoT()`); param renamed `tb` for thelper | `TestHelpers_AcceptTestingB` + lint 0 issues |
| Request options (new `options.go`) | `WithPath`, `WithHeader`, `WithLastEventID`, `WithDatastarSignals` (`?datastar=` query) — variadic on **all 5** Collect helpers | `options_test.go` (7 tests) + `options_internal_test.go` |
| New assertions | `RequireScript` (exact inner-JS match), `RequireEventID` (replay tests) | `assert_test.go` failure paths |
| Replay dogfood E2E | `TestE2E_ReplayWithLastEventID`: real `sse.EventStore` + `sse.Replay` driven by `WithLastEventID`, fresh-connect + reconnect cases | test green, `-race` |
| Coverage | **82.2% → 94.3%** (previously 0%: `RequireSignals`, `MustReadNEvents`; all Require* failure paths now covered via `recordingTB`) | `go test -cover` |
| Docs | New `datastartest/README.md`; doc.go sections (options, Ginkgo, script); AGENTS.md API table; FEATURES.md rows; CHANGELOG `[Unreleased]` | manual review |
| **Real bug fixed** | **Phantom events from dataless SSE frames**: heartbeat comments (`: heartbeat\n\n`) and id/retry-only frames dispatched empty `Event{}` values, violating the SSE spec (only data-bearing frames dispatch). Found by the new ssetest heartbeat dogfood test. Fixed via `dispatchFrame` + regression tests in both libraries. | `TestReadEvents_DatalessFramesNeverDispatch` (both repos) |
| Suite state | `go test ./... ./datastartest/... ./static/... -race` → all ok; `golangci-lint run` (all three modules) → **0 issues** | run at 2026-08-16 ~08:15 |

### go-sse — NEW `ssetest/` module (untracked, not yet committed)

The transport layer now has the same consumer-testing story as the protocol layer. Mirrors the datastartest architecture (separate Go module + `replace go-sse => ..`, so `testing` never leaks into production builds):

- **`reader.go`** — `ReadEvents`, `ReadNEvents`, `MustReadEvents`, `MustReadNEvents`; spec-correct dispatch (dataless frames never surface), CRLF-capable (via `bufio.ScanLines`).
- **`collect.go`** — `Collect`, `CollectWithRequest`, `CollectPost`, `CollectN`, `CollectWithTimeout` (all `testing.TB` + options), shared `doRequest` with `t.Cleanup(srv.Close)`.
- **`options.go`** — `WithPath`, `WithHeader`, `WithLastEventID`.
- **`assert.go`** — `RequireEventCount/Type/Data/DataContains/EventID/Retry`.
- **`search.go`** — `FindByType`, `FilterByType` (clone-then-delete; input not mutated).
- **`errors.go`** — `CodeSSEScanFailed` (`ssetest.sse_scan_failed`) via go-error-family (house pattern).
- **Tests** — reader edge cases, options propagation, assertion failure paths (`recordingTB`), `ExampleReadEvents`/`ExampleEventsString` with `// Output:`, `FuzzReadEvents` (11 seeds), `BenchmarkReadEvents`, and 5 dogfood E2E tests against **real go-sse features**: `Stream` field round-trip (type/id/retry/multi-line data), `Broadcaster` fan-out via `CollectN`, `Replay` + `WithLastEventID`, heartbeat invisibility, timeout-partial read.
- **State**: `go test ./... -race -cover` → ok, **96.9% coverage**. `go vet` clean. `go mod tidy` done.
- **Wiring**: `flake.nix` apps (test/test-race/build/vet/lint/coverage) now also run `cd ssetest && GOWORK=off ...`; CI jobs (test, vet, coverage, vulncheck, fuzz) gained `working-directory: ssetest` steps with `GOWORK: off`; lint job gained a second golangci-lint-action invocation for ssetest; hermeticCheck got a TODO comment for a future `hermeticCheckSsetest`.

---

## 2. PARTIALLY DONE / IN PROGRESS

- **go-sse docs wiring (todo #7) — flake + CI done, docs NOT started.** Missing: go-sse `CHANGELOG.md` `[Unreleased]` entry, `FEATURES.md` ssetest section, `AGENTS.md` module/commands update, root `README.md` consumer section, `ssetest/README.md` (datastartest got one; ssetest did not).
- **Todo #8 (final verification) partially done.** ssetest suite + vet green, but NOT yet run: `golangci-lint` on ssetest locally, full go-sse root suite after the flake/CI edits, `nix flake check` (flake.nix edited but never evaluated), actionlint/CI dry-run of `ci.yml`.
- **ssetest `go.mod` requires `go-sse v0.0.00010101000000-...`** (auto-generated pseudo-version from tidy+replace). Works locally; must be pinned to a real tag (e.g. `v0.5.0`) before publish — datastartest pins real versions and is the proven pattern.

---

## 3. NOT STARTED

1. `ssetest/README.md`.
2. go-sse CHANGELOG / FEATURES / AGENTS / README updates for ssetest.
3. Local golangci-lint run on ssetest (CI will hit it first otherwise).
4. `nix flake check` / `nix run .#test-race` end-to-end after flake edit.
5. erraudit pass over both changed modules (go-datastar CI enforces it; not run locally this session).
6. Release work: `ssetest/v0.1.0` + go-sse `v0.6.0` tag strategy; `datastartest/v0.3.0` cut.
7. `treefmt`/`nix fmt` pass (gofumpt + golines are stricter than the `gofmt -w` I used).
8. Root-module boundary guard for go-sse (mirror of go-datastar's `module_boundary_test.go`: root must never require ssetest).
9. Fuzz-seed additions for the dataless-frame bug (both corpora).
10. cqrs-htmx migration onto datastartest/ssetest (its tests hand-roll SSE parsing and use testify).

## 4. WHAT I MESSED UP OR RISKED (all caught and fixed)

- **sed rename bug**: `t.Errorf(` → mangled to `tb.Errorf"` (ate the paren) — caught by grep + build, repaired.
- **`recordingTB` nil-pointer panic**: embedded nil `testing.TB` promoted `Helper()` → SIGSEGV. Fixed with explicit no-op `Helper()`.
- **tagliatelle rename cascade**: `json:"lastID"` → `"lastId"` without updating the producer map key → test failed; fixed both sides.
- **Duplicate test name**: added `TestReadEvents_CRLFLineEndings` although one already existed (datastartest already handled CRLF — my "CRLF gap" claim was **wrong**; `bufio.ScanLines` strips `\r`). Compile error; removed the duplicate.
- **Botched python deletion** of that duplicate left orphaned statements (vet caught); repaired by hand.
- **Typo `fmt Sprint(i)`** in options_test.go — caught by LSP, fixed.
- **Dead code shipped**: `strings.TrimSuffix(text, "\r")` in ssetest `reader.go` is unreachable (ScanLines already strips it). Removed in datastartest; **still present in ssetest** — cosmetic cleanup pending.
- **Race in first draft of e2e tests**: broadcaster producer could fire before subscriber connected (Broadcast drops on no subscriber) → nondeterministic hangs; fixed with a repeat-broadcast producer loop.
- **CHANGELOG inaccuracy**: says 94.1%, final measured 94.3%.
- **Unverified CI wiring**: golangci-lint-action `working-directory` + config resolution from `ssetest/` (no `.golangci.yml` there — will fall back to defaults, diverging from repo config) never tested locally.

## 5. IMPROVEMENT IDEAS BEYOND THE SESSION'S SCOPE

- Property test: every `sse.Event` written by `WriteEvent` round-trips through `ssetest.ReadEvents` (id/retry/multi-line data).
- Consolidate the two near-identical parsers: have `datastartest` depend on `ssetest` (or accept deliberate duplication for module independence — needs a decision).
- `testing/synctest` for the timeout tests (remove real 150–200 ms sleeps).
- `coverage-gate` flake app: include ssetest (currently root-only, 90% threshold).
- Record `BenchmarkReadEvents` MB/s in FEATURES.md (like datastartest's ~131 MB/s note).
- `WithDatastarSignals` equivalent in ssetest (`WithQuery`) if transport consumers ask for it.
- Remove the stale go.work claim in go-sse AGENTS.md (no `/home/lars/projects/go.work` exists today — verified this session).

## 6. UP TO 50 NEXT STEPS (rough order)

1. Run `golangci-lint run ./...` inside `ssetest/` locally; fix findings.
2. Decide ssetest lint config: `ssetest/.golangci.yml` extending root, or CI `args: --config=../.golangci.yml`.
3. Run full go-sse root suite (`GOWORK=off GOEXPERIMENT=jsonv2 go test ./... -race`).
4. `nix flake check` after the flake.nix edit.
5. `nix fmt` (treefmt: gofumpt/golines) across both repos' new files; re-run tests.
6. Write `ssetest/README.md` (copy datastartest README structure).
7. Add go-sse `CHANGELOG.md` `[Unreleased]`: ssetest module + parser fix note.
8. Add `FEATURES.md` "Consumer Test Helpers (ssetest/)" section.
9. Update go-sse `AGENTS.md`: module table, commands (ssetest cd-invocations), conventions, TB invariant.
10. Update root `README.md` with a "Testing your handlers" section pointing at ssetest.
11. Pin `go-sse v0.5.0` (real tag) in ssetest `go.mod` require block.
12. Fix CHANGELOG 94.1% → 94.3% in go-datastar.
13. Remove dead `TrimSuffix` from ssetest reader.go.
14. Add dataless-frame + id-only/retry-only seeds to both fuzz corpora.
15. Run fuzz seed corpus locally (`go test ssetest -run FuzzReadEvents`).
16. Run erraudit on datastartest + ssetest (`--type-aware --enforce-go-error-family`).
17. Add `module_boundary_test.go` to go-sse root (root must not require ssetest).
18. Run `actionlint` on ci.yml (or eyeball-verify the added steps).
19. Push branch / let CI validate the workflow changes end-to-end.
20. Extend `coverage-gate` flake app with a ssetest threshold.
21. Write `hermeticCheckSsetest` buildGoModule derivation (TODO already in flake).
22. Decide + implement `WithQuery` for ssetest (or document path-embedded queries only).
23. Add `ExampleCollect`-style docs for ssetest (plain examples; helpers take TB).
24. Add `RequireDataJSON(t, evt, wantAny)` to ssetest (unmarshal-compare).
25. Add `BroadcastMany` fan-out e2e variant (batch path).
26. Add ssetest e2e for `SubscribeFilter` predicates.
27. Add ssetest e2e for `Shutdown` drain (CollectN + graceful close).
28. Add ssetest e2e for `OnDrop` (full-buffer drop observability).
29. Consider `testing/synctest` versions of timeout tests.
30. Run `BenchmarkReadEvents`, record MB/s in FEATURES.md.
31. WriteEvent↔ReadEvents round-trip property test in ssetest.
32. Tag `ssetest/v0.1.0` + cut go-sse `v0.6.0` (CHANGELOG, release notes, pkg.go.dev check).
33. Tag `datastartest/v0.3.0` (options + TB + parser fix).
34. Verify `go get github.com/larsartmann/go-sse/ssetest@v0.1.0` from a scratch module.
35. Cross-module compatibility matrix in go-sse README (go-sse ↔ ssetest ↔ go-datastar ↔ datastartest).
36. Migrate cqrs-htmx SSE/DataStar handler tests onto datastartest/ssetest.
37. Replace testify in cqrs-htmx/datastar with Ginkgo/Gomega per house policy (needs Lars' nod — it's banned per how-to-golang).
38. Document the phantom-frame fix as behavior change in datastartest release notes.
39. Add a Ginkgo usage example to both READMEs (GinkgoT() + Collect).
40. Update `example/datastar/VERIFY.md` with a replay-test recipe using WithLastEventID.
41. Consider `WithDatastarSignals` docs cross-link from go-datastar README testing section.
42. Sweep go-datastar AGENTS.md API table for `MustReadNEvents` (present but was 0%-covered — now covered).
43. Add `FindScript`/`FilterScripts` to datastartest if consumers index scripts often.
44. ssetest `Event.Data()` vs `DataLines` doc table in README.
45. Consider `Options` for `CollectWithTimeout` zero-event policy (fail vs empty slice).
46. Deprecation check: nothing deprecated this session; keep it that way.
47. `go work sync` idempotency check on go-datastar (CI enforces; deps unchanged, low risk).
48. govulncheck run over ssetest module.
49. Consider a `docs/guides/testing-your-sse-handlers.md` in go-sse.
50. Re-run this status report's checklist after the docs wiring lands.

## 7. QUESTIONS FOR LARS (max 3)

1. **Release cadence**: cut `ssetest/v0.1.0` + go-sse `v0.6.0` and `datastartest/v0.3.0` now (publish path: pin real versions in require blocks, tag nested modules), or let them ride untagged until the docs/CI verification steps land?
2. **Behavior-change policy**: the dataless-frame fix changes observable output (id/retry-only frames no longer produce events). Ship as a bug fix in `datastartest/v0.3.0`, or do you want a compat escape hatch?
3. **Duplication policy**: ssetest and datastartest now carry near-identical SSE parsers/collect cores. Consolidate (datastartest depends on ssetest) or keep deliberately duplicated for module independence? I kept them independent to avoid cross-repo release coupling at v0.x — confirm or overrule.

---

_Stopped here per instruction. Awaiting answers before executing Section 3/6 work._
