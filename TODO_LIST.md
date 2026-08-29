# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, see [ROADMAP.md](ROADMAP.md).
> Items are ranked by Pareto impact. Status is verified, not assumed.
> Completed work lives in [`CHANGELOG.md`](CHANGELOG.md), not here.

The 2026-08-29 SUPERB execution pass is complete: both go-sse releases are cut
and consumer-verified (root v0.6.0, ssetest v0.3.0), the go-datastar session
batch is landed behind its full gate, the datastartest v0.3.0 pairing is
verified healthy, actionlint+shellcheck gate the workflows, the flake-update
automation is dry-run-proven with auto-close hardening, and the report
conventions reached v1.1 with a template. Decision gates D1 (changelog policy)
and D2 (accept `38e79aa`) were resolved by their documented plan defaults; D3
(browser-E2E scope) stays open below.

## Status legend

| Status           | Meaning                                                                                       |
| ---------------- | --------------------------------------------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                                                                     |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                                                     |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed.                                       |
| ⚪ `WONT`        | Deliberately declined, with the reason inline. Revisit only if the trigger condition changes. |

## Correctness & safety

Nothing open — the `safeDropCall` batch, `OnDrop(nil)` tests, and the
`eventBrand.Name()` test shipped in the 2026-08-29 pass (root coverage 99.3%).
The CI-found `FuzzKeyedLines` property bug (multi-line keys) was fixed and its
crasher pinned the same day (`bdd08c2`).

## Parser parity & fuzz depth

| Status    | Item                                                  | Notes                                                                                                                                                                                                                                                                                            |
| --------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| ⚪ `WONT` | `testing/synctest` for the `CollectWithTimeout` tests | The `testing/synctest` guidelines prohibit network I/O inside a bubble, and every `Collect*` helper owns a real-socket `httptest` server. A fake-net rewrite would test a different transport than production. The cost is one ~200 ms test; keep it real. Source: 2026-08-29 execution pass.    |
| ⚪ `WONT` | Remaining gopls "unnecessary type argument" infos     | All inferable type arguments in tests were removed (2026-08-29). What remains is explicit by necessity: `NewBroadcaster[T]()` has no args to infer from and `WithBufferSize[T](…)`'s T appears only in its return type. `gopls check` CLI reports 0 diagnostics; the rest are editor-only hints. |
| ⚪ `WONT` | gopls `stdversion` friction on `encoding/json/v2`     | Root-caused: `jsonv2` std API is gated to a `go 1.27` directive in std metadata; the directive cannot exceed the 1.26.7 toolchain. The diagnostic is intrinsic until Go 1.27 and disappears then. Not a config problem; golangci-lint is clean.                                                  |

## CI & tooling

Nothing open. The weekly `nix flake update` cron workflow shipped and was
dry-run-verified (2026-08-29: no drift, all conditional steps correctly
skipped); it now auto-closes superseded `chore/flake-update-*` PRs before
opening a new one. `actionlint` runs on every push/PR with shellcheck presence
proven (run blocks are shellcheck-checked), and both tools live in the
devShell. CI's golangci-lint is pinned to the flake's version (v2.13.1) so
local and CI lint cannot drift again.

## Docs

Nothing open. Report conventions are at v1.1 (optional TL;DR, cross-repo cover
scope, specializes-the-global-format note) and `docs/status/_template.md` is
the copy-pasteable skeleton, linked from the conventions and AGENTS.md.

## Cross-repo

Nothing open. The go-datastar writer goldens landed (`a0c0aea`) behind that
repo's full gate (workspace race tests, `GOWORK=off` isolation ×3 modules,
erraudit ×3 with zero violations, `go work sync` idempotency — re-verified
2026-08-29 after the ssetest v0.3.0 release). `datastartest/v0.3.0` was tagged
by the concurrent session and scratch-consumer-verified from the proxy
(root + datastartest + go-sse resolve and compile together). Hygiene note for
its next release: the tagged `datastartest/go.mod` still carries local
`replace` directives — inert for consumers (dependency replaces are ignored)
but they should be dropped before tagging, as go-datastar's own checklist
prescribes.

## Blocked

| Status       | Item                                                        | Blocker                                                                                                                                                                                                                                                                                                                                                                           |
| ------------ | ----------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server) | Requires the browser-E2E scope decision first — see [docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md](docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md) (Option B vs C). The SUPERB plan's default (stay blocked) was applied 2026-08-29; the real DataStar JS client remains manually verified against the example server ([2026-08-05 archived report](status/archived/2026-08-05_10-15_datastar-example-cdn-url-fix.md)). |

## Declined

| Status    | Item                                        | Reason                                                                                                                                                                                                                                                     |
| --------- | ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ⚪ `WONT` | Rewrite the misleading auto-git commit `38e79aa` | SUPERB plan D2 default applied 2026-08-29: amending a month-old published commit violates the no-history-rewrite rule for a cosmetic gain. The AGENTS.md auto-git Gotcha documents the misleading message ("expand test coverage" describes a coverage-reducing deletion). Revisit only on an explicit user order — amend + force-with-lease. |
