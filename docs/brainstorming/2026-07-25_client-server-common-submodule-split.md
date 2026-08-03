# Brainstorming — 2026-07-25

_PRO/CONTRA on splitting go-sse into `client` / `server` / `common` Go sub-modules._

---

## Premise check (read this first)

The premise has a gap: **there is no client.** `doc.go`, `AGENTS.md`, and
`ROADMAP.md` ("Developer experience") all confirm the library is server-only;
a client-side `Dial` is explicitly a _future idea_ under that theme. So the
real question is **"pre-split now for a client that doesn't exist yet"** — and
the maintainer's own `fanOut` decision (ROADMAP "Parked decisions") already
shows they reject committing to API stability before a concrete use case
appears.

The honest framing: _is it worth paying multi-module tax today to prepare for a
client that may never arrive?_

---

## Current state

| Aspect                        | Value                                   |
| ----------------------------- | --------------------------------------- |
| go.mod files                  | **1** (`github.com/larsartmann/go-sse`) |
| Packages                      | **1** (`sse`, flat layout)              |
| Production source files       | ~6                                      |
| External deps                 | 2 (`go-branded-id`, `go-error-family`)  |
| `net/http` in production code | **only `stream.go`** — nowhere else     |
| Release                       | v0.2.0 (stable, tagged)                 |

That last row is the decisive finding. `event.go` (wire format), `fanout.go`
(fan-out), `replay.go`, and `constants.go` are **already `net/http`-free**. The
conceptual seams exist; they're just not enforced by separate `go.mod` files.

---

## Real consumer evidence (2026-07-25 audit)

Audited the 8 sibling projects a workspace graph listed as `[direct]` consumers
of `go-sse`. Ground truth is actual `.go` imports, not go.mod metadata:

| Project                        | go.mod      | Wire-format symbols                                                                                                                   | Server symbols                                                                                                         | Consumer type                                       |
| ------------------------------ | ----------- | ------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------- |
| cqrs-htmx                      | ✓ v0.2.0    | `Event`, `EventID`, `ContentType`, `EventConnected`, `EventHeartbeat`, `MustParseEventID`, `NewEventID`, `ParseEventID`, `WriteEvent` | `Stream`, `NewStream`, `SetHeaders`, `Broadcaster`, `NewBroadcaster`, `EventStore`, `Replay`, `LastEventIDFromRequest` | **Full-stack** (also re-exports go-sse as a facade) |
| go-workflow-auditlog           | (workspace) | `Event`, `ContentType`, `WriteEvent`                                                                                                  | —                                                                                                                      | **Wire-only**                                       |
| project-discovery-sdk          | (workspace) | —                                                                                                                                     | `NewStream`                                                                                                            | **Server-only**                                     |
| samber-do-auditlog             | ✓ v0.2.0    | `Event`, `ContentType`, `WriteEvent`                                                                                                  | —                                                                                                                      | **Wire-only**                                       |
| SwettySwipperWeb               | ✗           | —                                                                                                                                     | —                                                                                                                      | **Not a consumer**                                  |
| github-local-sync              | ✗           | —                                                                                                                                     | —                                                                                                                      | **Not a consumer**                                  |
| overview                       | ✗           | —                                                                                                                                     | —                                                                                                                      | **Not a consumer**                                  |
| projects-management-automation | ✗           | —                                                                                                                                     | —                                                                                                                      | **Not a consumer**                                  |

**4 real consumers, not 8.** The other 4 carry no `.go` import of `go-sse`.

### Decisive finding: the wire-only consumer is real

**2 of 4 consumers (50%) import `go-sse` purely for the wire format.**
`go-workflow-auditlog` and `samber-do-auditlog` both:

- run their own `http.Server` / `http.ServeMux` / `http.HandlerFunc`,
- do their own `w.(http.Flusher)` assertion,
- set `Content-Type` themselves via `w.Header().Set(..., sse.ContentType)`,
- call `sse.WriteEvent(w, sse.Event{...})` straight onto their
  `http.ResponseWriter`,
- and touch **none** of `Stream`, `NewStream`, `Broadcaster`, `fanOut`,
  `Replay`, `EventStore`.

Their `live/server.go` files are near-identical (the `handleSSE` /
`sendSnapshot` / `sendComplete` triplet) — a deliberate, repeated pattern in
this ecosystem of taking the wire format and _declining_ the server
scaffolding.

### What this refutes

CONTRA #3 below claimed _"Nobody imports `common` alone."_ That is **false**.
Two production services do exactly that today. The `common` seam has real,
independent customers — it is not hypothetical.

---

## What the split would look like

```
common/  → Event, EventID, ParseEventID, WriteEvent, WriteHeartbeat, WriteRetry,
           splitLines, ContentType, event-name constants
           deps: brandid, errorfamily, stdlib(io, fmt, strconv, strings)

server/  → Stream, Broadcaster, fanOut, Replay, EventStore, SetHeaders
           deps: common, net/http, sync, context, reflect, encoding/json/v2

client/  → EventSource / Dial / decoder   ← DOES NOT EXIST
           deps: common, net/http (client side)
```

The DAG is acyclic and clean — so a split is _technically trivial_. The
question is whether it earns its keep.

---

## PRO

1. **Role-focused API surface** — a client user sees only what they need; no
   `Broadcaster`/`Stream` noise. Real ergonomic win _if_ a client ever exists.
2. **Independent semver** — wire format (`common`) is frozen by the SSE spec
   (~unchanged in a decade); server-side can iterate on backpressure
   (ROADMAP section 1) without bumping client. Genuine versioning isolation.
3. **Compile-time boundary** — a client can't accidentally reach into
   `fanOut`/`Stream`. Today's flat package allows it.
4. **Stable base for the planned `Dial`** — extracting `common` now means the
   client lands on a frozen contract.

---

## CONTRA

1. **YAGNI / premature** — the biggest issue. Splitting for an imaginary
   consumer is the textbook anti-pattern ("premature generalization — don't
   build for imagined future needs"). You'd pay tax on a boundary with no user.
2. **Tiny project, huge overhead ratio** — 1 package, ~6 files vs. 3× `go.mod`,
   replace directives, `go.work`, per-module CI, version-sync discipline. The
   overhead dwarfs the codebase.
3. ~~**The `common` seam has zero composability payoff today**~~ — **REFUTED by
   the 2026-07-25 consumer audit.** Two production consumers
   (`go-workflow-auditlog`, `samber-do-auditlog`) import `go-sse` _only_ for the
   wire format (`Event` / `WriteEvent` / `ContentType`) and roll their own HTTP
   scaffolding around it. The "nobody imports `common` alone" claim was wrong;
   the seam has real independent customers. The residual point that survives:
   `common` still _co-releases_ with `server` today (one tag) — but that is a
   release-process fact, not evidence the seam is unwanted.
4. **No heavy deps to isolate** — the only non-stdlib weight is `net/http`,
   confined to `stream.go` and _already_ absent from the wire format. There's
   no transitive-dep bloat to fix. **(Still true, and it is the main reason the
   wire-only consumers pay ~zero practical cost today: `net/http` is stdlib.)**
5. **Release friction for a solo lib** — every wire-format tweak becomes a
   coordinated multi-module tag instead of one tag.
6. **Ergonomic regression** — `sse.NewBroadcaster[sse.Event]()` (one import)
   becomes `sseserver`/`ssewire` juggling for the full-stack task. The flat
   package is a feature. **(Note: this tax falls hardest on `cqrs-htmx`, the
   most intensive consumer, which uses the full stack AND re-exports it.)**

---

## Decision-framework score

Using the `go-modularize` skill's "When NOT to Modularize" rubric:

| Signal                             | Weight                   | For this project                                                                                                                                                                                                       |
| ---------------------------------- | ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Small project (<10 pkgs, 1 domain) | **High**                 | 1 package (unchanged)                                                                                                                                                                                                  |
| All packages change together       | ~~High~~ → **Medium**    | still one package (co-releases today), but the audit reveals divergent change drivers: wire is spec-frozen, server iterates (ROADMAP backpressure); audiences differ (2 wire-only vs. 1 server-only vs. 1 full-stack). |
| No external consumers of the seam  | ~~Medium~~ → **Refuted** | 2 of 4 consumers are wire-only                                                                                                                                                                                         |
| Prototype / spike                  | —                        | stable v0.2.0, not a spike                                                                                                                                                                                             |

**1 High + 1 Medium remain.** The "no lone consumers" signal collapsed under
the audit; the co-change signal weakened. The balance shifted toward "a seam
exists," but the _practical harm_ to wire-only consumers today is still ~zero
(`net/http` is stdlib; `brandid`/`errorfamily` are genuinely used by
`Event`/`WriteEvent`/`EventID`), which is what stays the hand from splitting now.

---

## Recommendation (revised 2026-07-25 with consumer evidence)

Two questions, two different answers:

**1. Three-way `client` / `server` / `common` split? — Still NO.**
No client exists (ROADMAP "Developer experience" lists `Dial` as a future
idea). Pre-splitting for a client that may never arrive remains premature.

**2. Two-way `common` / `server` split? — The case is now real, but not yet
urgent.** The audit proves the `common` seam has real customers (2 wire-only
consumers), and the spec-frozen wire format vs. iterating server is the classic
"stable core / volatile periphery" split. What still stays the hand:

- **No concrete harm today.** `net/http` is stdlib — wire-only consumers pay
  zero transitive cost. `brandid`/`errorfamily` are genuinely used by the wire
  path, so they are not waste either.
- **All 4 consumers are internal** to one maintainer, who bumps versions in
  lockstep trivially. Independent semver's value is marginal when the same
  person owns both sides of every seam.
- **The full-stack consumer (`cqrs-htmx`) pays an ergonomic tax** — one import
  becomes two for the most intensive user (which also re-exports the API).

Bottom line: **still don't split now**, but the original justification
("nobody imports `common` alone") is dead. The remaining blocker is purely the
absence of concrete harm, not the absence of a seam.

### Cheap hedge that costs nothing today (now battle-proven, not theoretical)

Keep `event.go`'s wire-format functions strictly `io.Writer`-based (they
already are) and don't let `net/http` leak into them. **Two production
services already depend on exactly this property** — the hedge is no longer a
prediction, it is a contract being exercised in the wild.

### Trigger criteria for revisiting (tightened)

Re-open this analysis when **any** of these becomes true:

- A concrete `client/` package is being written (not just roadmap-fantasized).
  ← still the strongest single signal.
- **The server gains a non-stdlib dependency** (e.g. a Redis event store per
  ROADMAP section 1) that the 2 wire-only consumers should never transitively
  pull in. ← this is now a _live_ trigger, not a hypothetical.
- A wire-only consumer asks to pin `common` and stop tracking server churn.
- A third wire-only consumer appears (2 → 3 turns a coincidence into an archetype).

Until one of those fires, the flat single-module layout remains the correct
shape — now with empirical confidence that the extraction will be mechanical
and welcomed when it comes.

---

## References

- `ROADMAP.md` — sections 1 (production readiness), 2 (developer experience / client Dial), 4 (parked decisions / module boundaries), 5 (raw ideas)
- `AGENTS.md` — "What This Library Is NOT", Broadcaster vs fanOut split
- `go-modularize` skill — Direction Neutrality, When NOT to Modularize
- Consumer audit (2026-07-25), verifiable at:
  `../cqrs-htmx/sse_{store,broadcaster,event}.go`, `../cqrs-htmx/ws_broadcaster.go`,
  `../go-workflow-auditlog/live/server.go`,
  `../project-discovery-sdk/daemon/server.go`,
  `../samber-do-auditlog/live/server.go`
