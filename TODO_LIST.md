# TODO List

> Short-term, actionable, bounded work items, verified against the actual code.
> For long-term vision and unrefined ideas, see [ROADMAP.md](ROADMAP.md).
> Items are ranked by Pareto impact. Status is verified, not assumed.
> Completed work lives in [`CHANGELOG.md`](CHANGELOG.md), not here.

State after the 2026-10-11 execution session (the SUPERB Pareto plan ran
end-to-end): ssetest v0.5.0 cut with its release page; v0.6.2 +
sseparse/v0.2.1 release pages backfilled; CI runners pinned to
ubuntu-24.04; the coverage gate moved to an up-only ratchet baseline
(100.0 ×3); verify.sh grew an AGENTS-facts assertion and a bare-shfmt
guard; AGENTS.md deep-pruned 49.4 → 27.0 KB; the release-page gap class
got a detector (release-verify step 4/4); every cross-repo batch
(go-datastar hygiene, sibling nolint migration, crush-config CI probe
hook, branching-flow README note, pdd pin verification, upstream BD001
issue) executed. Open work below is decisions and trigger-gated rows only.

## Status legend

| Status           | Meaning                                                                                       |
| ---------------- | --------------------------------------------------------------------------------------------- |
| 🔴 `TODO`        | Not started. Needs doing.                                                                     |
| 🟡 `IN_PROGRESS` | Actively being worked on.                                                                     |
| 🔵 `BLOCKED`     | Cannot proceed; external dependency or decision needed.                                       |
| ⚪ `WONT`        | Deliberately declined, with the reason inline. Revisit only if the trigger condition changes. |

## Open items

| Status    | Item                                                                   | Notes                                                                                                                                                                                                                                                                                                                                                                                                               |
| --------- | ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 `TODO` | Bump `minCorpusVectors` (25) at the next real WPT ingestion            | The corpus holds 29 vectors; the integrity floor should track its actual size. The ingestion generator's contract (idempotence, dup-skip, abort-on-mismatch) is pinned by `sseparse/gen_corpus_roundtrip_test.go`, so a bump cannot silently break the proof. Trigger-gated: no action until a new vector actually lands. Source: 05-49 report §f8.                                                                 |
| 🔴 `TODO` | Re-verify AGENTS.md's go-directive claims when the next root tag ships | Master is aligned at minor-form `go 1.27` ×3 (the deliberate fleet de-poisoning state — see the 0.6.2 CHANGELOG entry), and verify.sh's AGENTS-facts assertion now greps the claim against the tree on every run, so silent drift is dead; the remaining trigger is a future root tag whose committed directive could diverge from master. Part of the release checklist. Trigger-gated. Source: 07-03 report §f15. |

## Blocked (user decision needed)

Briefs for B1/B2/B5/B6 are in
[`docs/brainstorming/2026-10-11_blocked-decision-briefs.md`](docs/brainstorming/2026-10-11_blocked-decision-briefs.md)
— options + recommendation each, ready to decide in one sitting.

| Status       | Item                                                                     | Blocker                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| ------------ | ------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔵 `BLOCKED` | go-datastar v0.6.2: re-cut for green tag-CI, or accept frozen red        | The four v0.6.1 tags carry permanently red tag-CI (immutable old workflow: panicking lint pin + stale vendor hash). Release content was verified green pre-tag; master is green. Recommendation in the briefs: accept frozen red. Source: [07-03 report §g1](docs/status/archived/2026-09-29_07-03_ci-health-vendorhash-repair-pr-closure.md).                                                                                                                                                             |
| 🔵 `BLOCKED` | Policy for Actions-created PRs' `action_required` CI gate                | Weekly flake-update PRs sit in `action_required` until a human approves — they cannot gate themselves. PR #7 merged into a red master (2026-10-10). Recommendation in the briefs: keep manual approval + the never-merge-red rule as the actual guard. Source: 07-03 report §c1/§g2 + audit 2026-10-11.                                                                                                                                                                                                    |
| 🔵 `BLOCKED` | Dependabot-on-nix-hash policy                                            | Every dependabot PR that bumps a go.mod in a vendorHash-carrying module fails the nix job (dependabot cannot fix nix hashes). Standing options: auto-close-and-adopt on master, maintainer hash-fix push, or make the nix job non-blocking for dependabot branches. Currently handled ad hoc per PR. Source: 07-03 report §c2.                                                                                                                                                                             |
| 🔵 `BLOCKED` | buildflow: derive the normalize target from max(dep directives)          | An external normalize step lowered root+sseparse five times total (2026-09-29 ×2, 2026-10-01 ×3), fighting whatever floor master carried. The 2026-10-03 v0.6.2 de-poisoning removed the floor itself (minor form is now the policy), but the normalize step is still hardcoded and can diverge from the fleet policy again. The durable fix is buildflow's, not this repo's; verify.sh's directive-equality gate is the detector until then. Source: 07-03 report §c3/§f3 + 2026-10-01_17-05 report IMP4. |
| 🔵 `BLOCKED` | `branching-flow strong-id --exit-code` as a standing gate (verify.sh/CI) | Recommendation in the briefs: do not gate now (suppression set stable and small; a private-binary pin in the gate chain has already cost debugging sessions in other pins). Source: 2026-10-11_03-45 report §f8/§g2.                                                                                                                                                                                                                                                                                       |
| 🔵 `BLOCKED` | CI headless browser test (DataStar client + example server)              | Recommendation in the briefs: Option C (keep manual verification; the vendored-bundle drift class is structurally closed by go:embed). [docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md](docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md).                                                                                                                                                                                                                             |
| 🔵 `BLOCKED` | Buildcache rotation policy                                               | Host `/mnt/buildcache` (shared mount) hit 100% once, now 32% — cleanup/resize/rotation is owner territory. Nothing is currently blocked on it.                                                                                                                                                                                                                                                                                                                                                             |
