# Blocked-Decision Briefs — 2026-10-11

Four owner decisions from TODO_LIST Blocked, each with options and a
recommendation. Written by the 2026-10-11 execution session so the choices
can be made in one sitting. None blocks consumers today.

---

## B1 — go-datastar v0.6.2: re-cut for green tag-CI, or accept frozen red

**Situation.** The four v0.6.1-era tags carry permanently red tag-CI (an
immutable old workflow: a panicking lint pin + a stale vendor hash).
Release CONTENT was verified green pre-tag; their master is green; the
proxy `.mod` files were verified post-push.

| Option | Cost | Benefit |
| ------ | ---- | ------- |
| Re-cut (v0.6.3 ×3 + pages) | ~1 h + release churn + new version numbers consumers must parse | Green tag-CI on the release pages |
| Accept frozen red | 0 | None new — status quo |

**Recommendation: accept frozen red.** Tag-CI cosmetics for
content-verified releases are not worth three re-tags and the consumer-side
noise of new version numbers. If their release pages ever confuse a
consumer, add one line to each release page noting the red tag-CI is a
known immutable-workflow artifact — a comment edit, not a release.

---

## B2 — Policy for Actions-created PRs' `action_required` CI gate

**Situation.** Weekly flake-update PRs (GITHUB_TOKEN-authored) sit in
`action_required` until a human approves; they cannot gate themselves.
Two incidents: a 26-hour unapproved window (2026-08-31) and PR #7 —
approved late, merged into a then-red master, its merge-commit CI FAILED
(2026-10-10). Note the second failure was NOT the approval gap alone: the
rule "never merge a flake-update PR while master is red" (now in AGENTS.md)
is what would have stopped it.

| Option | Cost | Failure mode it leaves |
| ------ | ---- | ---------------------- |
| Approve per session | ~0 (habit) | Forgetting; approving into a red master |
| workflow_run self-gating | Workflow restructure + one-time debugging | Approval still manual; the PR's OWN evidence becomes pre-verified |
| Keep manual only | 0 | Both incidents' class remains human-dependent |

**Recommendation: keep manual approval, keep the never-merge-red rule as
the actual guard.** The workflow_run restructure is real work for a
once-a-week PR whose failure class (merge while red) is already gated by a
written rule + this session's habit. Revisit only if a third incident
lands despite the rule.

---

## B5 — `branching-flow strong-id --exit-code` as a standing gate (verify.sh/CI)

**Situation.** The strong-id suppression set is currently 3 stable,
documented rows (sseparse raw-ID fields). A gate would surface
deliberate-skip regressions, but adds another tool-version-sensitive pin
to the verify chain (like the golangci pin, but for a private binary).

| Option | Cost | Benefit |
| ------ | ---- | ------- |
| Add to verify.sh | One more pinned binary + version-skew surface | Suppression drift reddens immediately |
| Leave as periodic check | Relies on session discipline (now partially mechanized by the CI probe) | No new pin |

**Recommendation: do not gate now.** The suppression set has not drifted
since its creation, the branching-flow binary is private (no hermetic nix
story yet — a gate would need one), and every gate pin has already cost a
debugging session once (golangci, lint pin, vendor hashes). Revisit when
the suppression set grows past ~10 rows or after branching-flow ships a
public release with a stable CLI contract.

---

## B6 — CI headless browser test (DataStar client + example server)

**Situation.** The real DataStar JS client is manually verified against
the example server (`example/datastar/VERIFY.md`). The brainstorming doc
(`docs/brainstorming/2026-08-03_nix-vm-e2e-testing-with-chromedp.md`)
frames it as Option B (chromedp in this repo's flake) vs Option C (manual
verification, separate repo).

| Option | Cost | Benefit |
| ------ | ---- | ------- |
| B: chromedp in-repo | New transitive dep tree, flake browser plumbing, example-scoped value | Same browser behavior verified in CI |
| C: keep manual + separate repo | A browser-E2E repo to maintain | This library keeps zero browser-dep surface |

**Recommendation: Option C.** The examples are demonstrations, not
contract; ROADMAP explicitly rejects app-like expansion; and the vendored
JS (go:embed) means the client bundle cannot silently drift — the
drift class Option B would catch is already structurally closed. Revisit
if the examples grow into documented, consumer-copied patterns.

---

*Decisions B3 (dependabot-on-nix policy), B4 (buildflow normalize target),
and B7 (buildcache rotation) stay owner/host-only — no brief needed; B4's
durable fix lives in buildflow (tracked there), B3's per-PR ad hoc handling
continues, B7 is a host-mount task.*
