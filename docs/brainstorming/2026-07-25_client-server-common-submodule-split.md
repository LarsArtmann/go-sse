# Brainstorming — 2026-07-25

_PRO/CONTRA on splitting go-sse into `client` / `server` / `common` Go sub-modules._

---

## Premise check (read this first)

The premise has a gap: **there is no client.** `doc.go`, `AGENTS.md`, and
`ROADMAP.md:23` all confirm the library is server-only; a client-side `Dial` is
explicitly a *future idea* under "Developer experience." So the real question is
**"pre-split now for a client that doesn't exist yet"** — and the maintainer's
own `fanOut` decision (`ROADMAP.md:47`) already shows they reject committing to
API stability before a concrete use case appears.

The honest framing: *is it worth paying multi-module tax today to prepare for a
client that may never arrive?*

---

## Current state

| Aspect | Value |
| --- | --- |
| go.mod files | **1** (`github.com/larsartmann/go-sse`) |
| Packages | **1** (`sse`, flat layout) |
| Production source files | ~6 |
| External deps | 2 (`go-branded-id`, `go-error-family`) |
| `net/http` in production code | **only `stream.go`** — nowhere else |
| Release | v0.2.0 (stable, tagged) |

That last row is the decisive finding. `event.go` (wire format), `fanout.go`
(fan-out), `replay.go`, and `constants.go` are **already `net/http`-free**. The
conceptual seams exist; they're just not enforced by separate `go.mod` files.

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

The DAG is acyclic and clean — so a split is *technically trivial*. The
question is whether it earns its keep.

---

## PRO

1. **Role-focused API surface** — a client user sees only what they need; no
   `Broadcaster`/`Stream` noise. Real ergonomic win *if* a client ever exists.
2. **Independent semver** — wire format (`common`) is frozen by the SSE spec
   (~unchanged in a decade); server-side can iterate on backpressure
   (ROADMAP theme 1) without bumping client. Genuine versioning isolation.
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
3. **The `common` seam has zero composability payoff today** — litmus test:
   *"if every consumer imports A and B together, the seam earns nothing."*
   Every consumer of `common` also imports `server` (or a future client).
   Nobody imports `common` *alone*. And `common` co-changes 100% with `server`
   — `Event`/`WriteEvent` ARE the server's core.
4. **No heavy deps to isolate** — the only non-stdlib weight is `net/http`,
   confined to `stream.go` and *already* absent from the wire format. There's
   no transitive-dep bloat to fix.
5. **Release friction for a solo lib** — every wire-format tweak becomes a
   coordinated multi-module tag instead of one tag.
6. **Ergonomic regression** — `sse.NewBroadcaster[sse.Event]()` (one import)
   becomes `sseserver`/`ssewire` juggling for the 99% server-side task. The
   flat package is a feature.

---

## Decision-framework score

Using the `go-modularize` skill's "When NOT to Modularize" rubric:

| Signal | Weight | For this project |
| --- | --- | --- |
| Small project (<10 pkgs, 1 domain) | **High** | 1 package |
| All packages change together | **High** | nothing left TO diverge |
| No external consumers (of a client) | Medium | client doesn't exist |
| Prototype / spike | — | stable v0.2.0, not a spike |

**2 High + 1 Medium → "consider partial modularization,"** but the only partial
candidate (`common`) has no payoff because it never moves independently of
`server`. The score points at a seam that the seam itself cannot justify.

---

## Recommendation

**Don't split now.** Keep the single package. The right trigger is the day a
real `client/Dial` lands — *then* extract `common` in one mechanical pass,
because the seam (`event.go`, already `net/http`-free) is already clean. No
rework, no regret.

### Cheap hedge that costs nothing today

Keep `event.go`'s wire-format functions strictly `io.Writer`-based (they
already are) and don't let `net/http` leak into them. That preserves a
zero-friction future extraction without paying multi-module tax on a library
that hasn't needed it.

### Trigger criteria for revisiting

Re-open this analysis when **any** of these becomes true:

- A concrete `client/` package is being written (not just roadmap-fantasized).
- An external consumer asks to import the wire format without `net/http`.
- The server gains a heavy dependency (e.g. Redis store) that a client should
  never transitively pull in.

Until then, the flat single-module layout is the correct shape.

---

## References

- `ROADMAP.md` — themes 1 (production readiness), 2 (dev experience, client Dial)
- `AGENTS.md` — "What This Library Is NOT", Broadcaster vs fanOut split
- `go-modularize` skill — Direction Neutrality, When NOT to Modularize
