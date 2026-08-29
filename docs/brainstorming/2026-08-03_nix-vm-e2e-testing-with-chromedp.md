# Nix-based E2E testing for the DataStar example

> **Status:** Idea — needs a scope decision before any implementation.
> **Raised:** 2026-08-03
> **Related:** TODO_LIST.md ("CI headless browser test", 🔵 BLOCKED), ROADMAP.md non-goals.

## The question

Can we use Nix VMs for proper end-to-end testing of the DataStar example, e.g. via `chromedp`?

## Short answer

**Yes, feasible — but a full Nix VM is the wrong tool for this repo.** A
headless-Chromium `check` (no VM) is the right shape _if_ we decide browser E2E
belongs in this library's flake at all. That decision is blocked on a scope
call (see [Open questions](#open-questions)).

## What I found in the repo

- **go-sse is a pure library.** `flake.nix:45-49` states `buildGoModule` is used
  _solely_ as a hermetic compile+test check; no binary is published. Adding a
  browser E2E check is a category shift from "does the library compile and pass
  unit tests" to "does the example render in a browser."
- **The example loads DataStar from a CDN at runtime**
  (`example/datastar/main.go:104` → `cdn.jsdelivr.net/gh/starfederation/datastar@...`).
  A hermetic Nix check has **no network**, so this is the hard blocker for any
  offline browser test — VM or not.
- **`chromedp` is not a dependency.** go.mod has only the two `larsartmann/*`
  libs. Adding chromedp pulls in a transitive dependency tree.
- **The TODO_LIST already tracks this** as 🔵 BLOCKED: "CI headless browser
  test … Requires the real-client verification above first."
- **ROADMAP non-goals** explicitly reject expanding toward app-like behavior
  (dashboard/routes/templates). Browser E2E of an _example_ is adjacent to
  that line.

## The three options

| Option                                               | Verdict                            | Rationale                                                                                                                                                                                                                                                       |
| ---------------------------------------------------- | ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **A. NixOS VM test** (`pkgs.testers.vmTests` / QEMU) | ❌ Overkill                        | Boots a full OS (~10–30s+) to verify an SSE _example_. Disproportionate for a transport library; bloats `nix flake check` time and the closure by hundreds of MB. Only justified when you need to test kernel/systemd-level behavior, which is irrelevant here. |
| **B. Headless Chromium + chromedp in a Nix `check`** | ✅ Recommended (if we do E2E here) | `pkgs.chromium` + a Go test using `chromedp.NewExecAllocator`. No VM — chromedp speaks CDP directly (no chromedriver needed). Fast, hermetic via Nix. This is the standard Go web E2E pattern.                                                                  |
| **C. Skip browser E2E entirely in this flake**       | ⚠️ Defensible                       | The library's correctness lives at the wire-format level (already heavily tested via `httptest`). A browser test verifies the _DataStar example_, i.e. documentation, not the product. E2E could live in a separate example-verification repo or stay manual.   |

### Why not the VM (Option A)

A NixOS VM test is the heaviest hammer in the Nix toolbox. It exists for
verifying system services, multi-machine interaction, systemd units, and
kernel-level behavior. None of that is in scope for an SSE transport library.
Using it to load a webpage in a browser would be a trophy-case check: expensive
to run, expensive to maintain, and verifying the wrong layer. The headless
chromedp path (Option B) verifies the exact same browser behavior at a fraction
of the cost.

### The CDN blocker (applies to A and B)

Both browser-based options require the DataStar client JS to be available
without network access. Two ways to solve it:

1. **Vendor the bundle** — commit `bundles/datastar.js` into the example
   directory (or a `vendor/` dir), reference it locally instead of the CDN.
   Simplest; makes the example reproducible offline for humans too.
2. **Nix fixed-output fetch** — `pkgs.fetchurl` with a pinned hash, served by
   the test harness. Keeps the repo smaller but couples the example's HTML to
   the test setup.

Option 1 is cleaner because it also fixes the example for anyone reading the
repo offline.

## Open questions (blocking)

These are scope/tradeoff decisions, not implementation details:

1. **Does browser E2E belong in this library's flake?** (Option B vs C)
   - B: keeps verification co-located with the code; increases CI time and
     adds a heavy dependency (Chromium) to the flake.
   - C: respects the "pure library" boundary; E2E moves to a separate
     example-verification repo or stays manual.
2. **If B: how to handle the DataStar CDN dependency?** Vendor the JS bundle
   into the repo (recommended) or fetch via Nix fixed-output derivation?
3. **Does adding `chromedp` to `go.mod` violate the minimal-dependency
   principle?** The library currently has only two deps, both `larsartmann/*`.
   chromedp would be the first external dependency. It could live in a separate
   `go.mod` (e.g. under `example/e2e/`) to keep the library's module graph
   clean — but that adds a nested-module maintenance cost.

## If we proceed with Option B, the steps would be

1. Vendor the DataStar JS bundle into the example (or Nix-fetch it).
2. Update the example HTML to reference the local bundle instead of the CDN.
3. Add `chromedp` as a test-only dependency (ideally in a nested `go.mod` under
   `example/e2e/` to avoid polluting the library's module graph).
4. Write a Go E2E test: start the example server on a random port, launch
   headless Chromium via `chromedp.NewExecAllocator`, navigate to `/`, wait for
   `#status` to show "Complete!", assert `$progress` reached 100.
5. Wire a `checks.e2e` into `flake.nix` using `buildGoModule` (or a custom
   derivation) that runs the E2E test with `pkgs.chromium` available.
6. Update TODO_LIST: unblock "CI headless browser test" and mark the
   real-client verification as done.

## Recommendation (tentative)

**Option B with a nested `go.mod`** — but only after confirming that browser
E2E of an example is worth the ongoing CI cost for a pure transport library.
If the answer is "rarely," Option C (manual verification, separate repo) is the
honest choice and keeps this flake fast and focused.
