# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, see [ROADMAP.md](ROADMAP.md).
> Items are ranked by Pareto impact. Status is verified, not assumed.
> Completed work lives in [`CHANGELOG.md`](CHANGELOG.md), not here.

## Status legend

| Status           | Meaning                                                                                      |
| ---------------- | -------------------------------------------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                                                                    |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                                                    |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed.                                      |

## Correctness & safety

| Status    | Item                                                                          | Notes                                                                                                                                                                                     |
| --------- | ----------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Add `safeDropCall` panic recovery for `onDrop`                                 | Predicates get `safePredCall` (`fanout.go:241`); a panicking `onDrop` callback still crashes `Broadcast`/`BroadcastMany` on the broadcasting goroutine. Asymmetric with the predicate contract. Source: `docs/status/archived/2026-08-13_02-29_on-drop-review-fixes-and-self-critique.md` GAP 1. |
| 🔴 `TODO` | Document the `onDrop` re-entrancy constraint + cross-reference it from `Broadcast`/`BroadcastMany` | The callback fires inside the fan-out read lock (`sendAllLocked`); calling back into the broadcaster (e.g. `OnDrop` setter) from it deadlocks. Neither `WithOnDrop`/`OnDrop` doc comments nor `Broadcast`/`BroadcastMany` mention the constraint or the observability hook. Source: 2026-08-13 report GAP 2 / §b. |
| 🔴 `TODO` | Pin `OnDrop(nil)` / `WithOnDrop(nil)` clear-callback behavior with tests      | Both doc comments say "pass nil to clear"; no test exercises it (`drop_test.go` has registration and fire tests only). Source: 2026-08-13 report §f 12–13.                                                                |
| 🔴 `TODO` | Add a direct test for `eventBrand.Name()` (`event.go:17`) — 0% covered since `TestEventID_StringIncludesBrandName` was deleted 2026-07-27 | The method is go-sse's own code; the deleted test asserted unreleased upstream `String()` behavior. Needs an in-package test (symbol is unexported; suite convention is external `sse_test`). Source: `docs/status/2026-07-27_10-26_removed-brand-name-test-and-self-review.md` §IMP1. Verified 2026-08-29: `go tool cover -func` shows `Name 0.0%`. |

## Parser parity & fuzz depth

| Status    | Item                                                                                     | Notes                                                                                                                                                           |
| --------- | ---------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Port the ssetest fuzz regression seeds to `datastartest` (go-datastar)                    | ssetest pins the trailing-LF regression (`testdata/fuzz/FuzzWriteReadRoundTrip/2ba7b6a0aaf94e65`) plus WPT seed vectors; `datastartest/testdata/fuzz/` is empty. The two parsers are deliberately duplicated and must stay in lockstep (AGENTS.md). |
| 🔴 `TODO` | Commit the interesting-input fuzz corpus growth + the `"0data: hello\n\n"` crasher as explicit seeds | 34–53 new interesting inputs from the 2026-08-16 bursts live only in local build caches; the crasher that exposed the substring-assertion bug is only referenced in a comment (`ssetest/reader_fuzz_test.go:108`). Sources: `docs/status/archived/2026-08-16_09-04…` §f 3–4. |
| 🔴 `TODO` | Add the missing fuzz targets to CI: `FuzzKeyedLines` (root), `FuzzWriteReadRoundTrip` (ssetest) | `.github/workflows/ci.yml` fuzz job runs only `FuzzWriteEvent`, `FuzzParseEventID`, `FuzzReadEvents`; the round-trip property is the conformance contract and never fuzzes in CI. Verified 2026-08-29. |
| 🔴 `TODO` | Fuzz `splitSSELines` directly + property: writer `splitLines` ≡ reader `splitSSELines` line boundaries | The SplitFunc has corpus coverage but no dedicated fuzz target; terminator equivalence between writer and reader is unpinned. Source: `docs/status/archived/2026-08-16_10-14…` §f 23–24. |
| 🔴 `TODO` | Property-test `KeyedLines`/`SendKeyed` round-trip through the wire                       | Only exercised via examples and unit fixtures today. Source: `docs/status/archived/2026-08-16_11-58…` §f 18.                                                              |
| 🔴 `TODO` | Explicit BOM-split-across-reads matrix test (BOM bytes at every chunk boundary)           | Covered only implicitly by the 1–4096 chunk-size sweep. Source: 2026-08-16_11-58 §f 19.                                                                          |
| 🔴 `TODO` | Add a sticky-ID reconnect assertion to an E2E test (real HTTP + `Last-Event-ID`)         | The e2e suite predates the sticky-id semantics; no end-to-end test pins that the browser-visible `id:` survives a reconnect round-trip. Source: 2026-08-16_10-14 §f 25–26. |

| 🔴 `TODO` | Run `nix flake check --all-systems` (darwin/aarch64 eval) at least once           | Both modules are pure Go; cross-system eval is cheap insurance. Source: `docs/status/2026-07-23_21-35…` §f.46. |
| 🔴 `TODO` | Pin `govulncheck` in CI (`@latest` → tagged version)                              | The lint job documents why `latest` is unacceptable one job above; vulncheck repeats the anti-pattern. Source: `docs/status/2026-07-26_18-52…` §d.4.   |
| 🔴 `TODO` | Align ssetest `go.mod` go-directive (1.26.6) with root (1.26.7) — bundle with the next ssetest change | Recomputing `vendorHashSsetest` is the cost; pay it once, not per-commit. |

## Coverage

| Status    | Item                                                                                    | Notes                                                                                                                |
| --------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Cover the 8 `resp.Body.Close()` error branches with an erroring `io.ReadCloser` fake     | Restores ssetest ~97% / closes the honest regression from the erraudit fix. Source: `docs/status/archived/2026-08-16_09-04…` §f 2. |

## CI & tooling

| Status    | Item                                                                                  | Notes                                                                                                                                 |
| --------- | -------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| 🔴 `TODO` | `scripts/verify.sh` (fmt + lint + test + flake check) as the one-command pre-push ritual, or an equivalent pre-push hook | Kills the auto-git daemon's unverified-push failure class. Source: 2026-08-16_11-58 §f 10, 15. |
| 🔴 `TODO` | CI: build the examples (`go build ./example/...`) and add a `templ generate` staleness check | CI compiles only root + ssetest; a broken example or stale `index_templ.go` ships unnoticed. Sources: 2026-08-07_00-44_htmx §f 30, 2026-08-06_22-23 §c. |
| 🔴 `TODO` | CI: run `nix flake check` (or a hermetic-build equivalent) as a required check          | The hermetic gate that caught the vendorHash split has never run in CI. Source: 2026-08-16_11-58 §f 11. |
| 🔴 `TODO` | Extend the `coverage-gate` flake app with an ssetest threshold                          | Gate covers the root module only. Source: 2026-08-16_08-23 §6.20. |
| 🔴 `TODO` | `testing/synctest` for the `CollectWithTimeout` tests (drop the real 150–200 ms sleeps) | Source: 2026-08-16_09-04 §f 9. |
| 🔴 `TODO` | gopls hygiene: resolve ~17 "unnecessary type argument" infos in tests; root-cause the `GOEXPERIMENT=jsonv2` stdversion diagnostic friction | Cosmetic but nags every session. Sources: 2026-08-16_11-58 §f 21, 2026-08-16_09-04 §f 6. |

## Docs

| Status    | Item                                                                                  | Notes                                                                 |
| --------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| 🔴 `TODO` | Write `docs/guides/reconnection-and-retry.md` — the 5-layer retry model (browser `EventSource` → `retry:` hint → `Last-Event-ID`+`Replay` → `Heartbeat` → caller-level `Shutdown` retry) | Exists only in chat; the go-retry decline decision references it. Source: `docs/status/archived/2026-08-07_08-41…` §B3. |
| 🔴 `TODO` | Add `RequireDataJSON(tb, evt, want any)` to ssetest (unmarshal-and-compare assertion)    | JSON-payload handlers currently hand-roll unmarshal in tests. Source: 2026-08-16_08-23 §6.24. |
| 🔴 `TODO` | datastartest parity batch (go-datastar repo): conformance README section, WPT writer goldens, chunk-boundary fuzz target | Tracked here so the two-repo parser contract stays visible; work lands in the go-datastar repo. Sources: 2026-08-16_10-14 §b, 2026-08-16_11-58 §f 2–3, 17. |
| 🔴 `TODO` | Add a release checklist to `CONTRIBUTING.md` (hermetic gate green, FEATURES/ROADMAP refreshed, tag validated in a worktree, `gh release create` staged, push) | Zero release guidance today; the first three releases fumbled the same steps. Sources: 2026-07-24_03-12 §e, 2026-07-26_19-48 §f.7. Verified 2026-08-29: 0 "release" matches in CONTRIBUTING.md. |

## Blocked

| Status       | Item                                                        | Blocker                                           |
| ------------ | ----------------------------------------------------------- | ------------------------------------------------- |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the browser-E2E scope decision first — see [docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md](docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md) (Option B vs C). The real DataStar JS client was manually verified against the example server on 2026-08-05 (`docs/status/archived/2026-08-05_10-15_datastar-example-cdn-url-fix.md`). |
