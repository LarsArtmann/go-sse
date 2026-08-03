# Brainstorming — 2026-08-03

_Could samber/do v2 (as an optional submodule) provide better Shutdown and Health checks for go-sse consumers?_

---

## Premise

The question: **"if we would add samber/do v2 (maybe as an optional submodule)
could we provide better Shutdown and Health checks to endusers?"**

Short answer: **yes for orchestration convenience, no for new capability.**
go-sse already has the shutdown primitive (`Broadcaster.Close()`); what it
lacks is a _drain_ (wait for in-flight sends) and a _health signal_. samber/do
would wire those into a container's lifecycle — but the primitives themselves
belong in core, not behind a DI dependency.

---

## What samber/do v2 actually provides

Verified against [pkg.go.dev/github.com/samber/do/v2](https://pkg.go.dev/github.com/samber/do/v2) (v2.1.0):

### Lifecycle interfaces

```go
type Healthchecker interface               { HealthCheck() error }
type HealthcheckerWithContext interface     { HealthCheck(context.Context) error }
type Shutdowner interface                  { Shutdown() }
type ShutdownerWithError interface         { Shutdown() error }
type ShutdownerWithContextAndError interface { Shutdown(context.Context) error }
```

### Injector-level batch methods

```go
func (s *RootScope) Shutdown() *ShutdownReport
func (s *RootScope) ShutdownWithContext(ctx context.Context) *ShutdownReport
func (s *RootScope) HealthCheck() map[string]error
func (s *RootScope) ShutdownOnSignals(signals ...os.Signal) (os.Signal, *ShutdownReport)
```

The three value propositions for go-sse:

1. **`injector.Shutdown()`** — one call drains ALL registered broadcasters.
   Without it, the consumer manually tracks and closes each broadcaster.
2. **`injector.HealthCheck()`** — returns aggregate health from all services.
   A Broadcaster health check could report "closed? subscriber count? drop rate?".
3. **`injector.ShutdownOnSignals(SIGTERM, SIGINT)`** — the graceful-shutdown
   helper from TODO_LIST, provided out of the box by the container.

---

## What go-sse has today

| Capability | Current state | Gap |
| ---------- | ------------- | --- |
| Close all subscribers | `Broadcaster.Close()` — instant, no drain | No context/deadline; no "wait for in-flight sends" |
| Health signal | `Broadcaster.SubscriberCount()` (count only) | No "am I closed?" query; no drop counter; no structured status |
| Signal handling | Consumer's responsibility (TODO_LIST item) | No helper exists |
| Per-subscriber observability | `OnSubscribe` / `OnUnsubscribe` hooks | No aggregate metrics (total broadcasts, total drops) |

**Key insight:** the gaps are in go-sse's own API surface, not in DI wiring.
samber/do would not add a drain method or a health struct — it would only
_call_ them. The library must provide the primitives first.

---

## Three architectural options

### Option A — samber/do as a hard dependency in core

```go
// broadcaster.go
type Broadcaster[T any] struct {
    *fanOut[T]
    injector do.Injector  // ← added
}
```

**Rejected.** go-sse has 2 dependencies (`go-branded-id`, `go-error-family`),
both same-author utility modules. samber/do pulls in `testify/testify` and a
full DI container. Adding it to core violates:
- The ROADMAP non-goal: "no framework or payload-format opinions."
- The wire-only consumer contract (2 of 4 consumers import go-sse for the wire
  format only — they do not want a DI container transitively).
- The brainstorming doc's conclusion: `net/http` is stdlib, so the flat layout
  costs wire-only consumers ~zero today. samber/do would end that.

### Option B — optional `di/` subpackage with thin adapters

```
di/
  adapters.go   ← broadcasterAdapter, streamAdapter implementing do.* interfaces
```

```go
package di

func RegisterBroadcaster[T any](injector do.Injector, b *sse.Broadcaster[T]) {
    do.ProvideValue(injector, &broadcasterAdapter[T]{b: b})
}

type broadcasterAdapter[T any] struct{ b *sse.Broadcaster[T] }

func (a *broadcasterAdapter[T]) Shutdown(ctx context.Context) error {
    return a.b.Shutdown(ctx)  // delegates to core drain method (TODO_LIST item)
}

func (a *broadcasterAdapter[T]) HealthCheck(ctx context.Context) error {
    return a.b.Health().Error()  // delegates to core health method (proposed)
}
```

**Pros:**
- Only consumers using samber/do import it (`samber-do-auditlog` uses both today).
- Core stays clean — no DI dependency.
- Follows the `datastar/` optional-subpackage precedent (already scaffolded).

**Cons:**
- Sets a precedent: "every Go library needs a `di/` subpackage for every DI container."
- The adapter is 15-30 lines — consumers can write it themselves in their
  composition root, where DI integration belongs.
- samber/do's own docs show the pattern is to implement lifecycle interfaces on
  YOUR types, not wrap library types.

### Option C — core primitives + documented pattern (recommended)

1. **Core go-sse adds the primitives** (independent of any DI container):

```go
// Graceful drain — respects context deadline, unlike instant Close()
func (b *Broadcaster[T]) Shutdown(ctx context.Context) error

// Structured health status
type BroadcasterHealth struct {
    Closed          bool
    SubscriberCount int
    // future: TotalBroadcasts, TotalDrops (needs counters)
}
func (b *Broadcaster[T]) Health() BroadcasterHealth
```

2. **An example** in `example/` or `doc.go` shows the samber/do integration:

```go
// In the consumer's composition root:
type SSEService struct{ b *sse.Broadcaster[sse.Event] }

func (s *SSEService) Shutdown(ctx context.Context) error { return s.b.Shutdown(ctx) }
func (s *SSEService) HealthCheck(context.Context) error {
    if s.b.Health().Closed { return errors.New("broadcaster closed") }
    return nil
}

// Register with the container:
do.Provide(injector, func(i do.Injector) (SSEService, error) {
    return SSEService{b: sse.NewBroadcaster[sse.Event]()}, nil
})
```

3. **No `di/` subpackage shipped.** The consumer writes 10 lines of adapter in
   their own composition root. This is the samber/do-canonical pattern.

---

## Recommendation: Option C

| Criterion | Option A (hard dep) | Option B (`di/` subpackage) | Option C (primitives + example) |
| --------- | ------------------- | --------------------------- | ------------------------------- |
| Core stays clean | ❌ | ✅ | ✅ |
| No transitive deps for wire-only consumers | ❌ | ✅ | ✅ |
| Consumer gets orchestration convenience | ✅ | ✅ | ✅ (10 lines) |
| Consumer controls DI integration | ❌ | ❌ (library's adapter) | ✅ |
| No "di/ per container" precedent | ✅ | ❌ | ✅ |
| Core API gains drain + health (works without DI) | ❌ | ❌ | ✅ |

Option C wins on every criterion. The primitives (`Shutdown(ctx)`, `Health()`)
are valuable to ALL consumers — not just samber/do users. A consumer using
`fx`, `wire`, or manual construction gets the same drain and health
capabilities. The samber/do integration is 10 lines of adapter code the
consumer writes where it belongs: their composition root.

### What this means for TODO_LIST

The "Graceful-shutdown helper" TODO should evolve from "drain subscribers on
SIGTERM" to two bounded items:

1. **`Broadcaster.Shutdown(ctx context.Context) error`** — drain respecting
   context deadline (the real primitive).
2. **`Broadcaster.Health() BroadcasterHealth`** — structured status query.

The signal-handling loop stays in the consumer (or samber/do's
`ShutdownOnSignals`). go-sse provides the drain; the consumer wires it to
whatever lifecycle manager they use.

### What this means for the empty `datastar/` directory

The `datastar/` directory is currently empty (scaffolded but not built). It
sets a precedent for optional subpackages. If we later decide Option B is
worth it (e.g., `samber-do-auditlog` asks for it), the `di/` subpackage can be
added without touching core. But the trigger should be a concrete consumer
request, not proactive library design.

---

## Trigger criteria for revisiting

Re-open Option B (ship a `di/` subpackage) when **any** of these fires:

- A concrete consumer (`samber-do-auditlog` or other) asks for a go-sse-provided
  adapter instead of writing their own.
- The adapter grows beyond 30 lines (e.g., needs scoped shutdown, health
  aggregation across multiple broadcasters, or metric collection).
- samber/do adds a feature that requires library-level integration (not just
  interface implementation).

Until then, Option C (primitives + documented pattern) is the correct shape.

---

## References

- [pkg.go.dev/github.com/samber/do/v2](https://pkg.go.dev/github.com/samber/do/v2) — API verified v2.1.0
- `ROADMAP.md` — section 1 (production readiness), section 5 (raw ideas)
- `TODO_LIST.md` — "Graceful-shutdown helper" (the drain primitive this analysis depends on)
- `docs/brainstorming/2026-07-25_client-server-common-submodule-split.md` — wire-only consumer evidence (2 of 4 consumers), module boundary principles
- `samber-do-best-practices` skill — canonical patterns, DO-1 → DO-6 anti-patterns
