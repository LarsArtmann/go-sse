# Should go-sse adopt `go-retry`?

**Date:** 2026-08-07
**Subject:** [`github.com/larsartmann/go-retry`](https://github.com/LarsArtmann/go-retry) v0.1.0
**Verdict:** **No** — not in the core library. Re-open only when a client `Dial`
helper is actually being written (ROADMAP §2).

---

## 1. What `go-retry` is

A ~140-line retry loop: `Do(ctx, Config, AttemptFunc)` with exponential backoff,
additive jitter (up to 50%), a pluggable `IsRetryable` predicate, and
`OnRetry`/`OnExhausted` hooks. Errors are `go-error-family` `Infrastructure`
sentinels (`retry.ErrExhausted`, `retry.ErrCanceled`). Its only non-stdlib
dependency is `go-error-family`.

## 2. PRO — the honest case for adoption

These are real, and were verified rather than assumed.

| #  | Argument                                                                                                                                                                                                                                                      | Evidence                                                                                |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| P1 | **Zero new transitive modules.** `go-retry`'s only dependency is `go-error-family v0.10.0`, which go-sse already requires at exactly that version. Adopting adds one line to `go.mod` and no new packages to the build graph.                                 | `go list -deps github.com/larsartmann/go-retry` yields only stdlib + `go-error-family`. |
| P2 | **Ecosystem consistency.** Same author, same error family, same stable-code convention (`retry.exhausted`) that go-sse already follows (`sse.write_failed`). No impedance mismatch in error handling.                                                         | `errorfamily.Wrapf` used identically in both.                                           |
| P3 | **Correct home for the logic _if_ go-sse ever ships a client.** A `Dial` helper reconnecting an `EventSource` is the textbook retry-with-backoff use case, and `Event.Retry` (the server's suggested reconnect interval) would feed `InitialDelay` naturally. | ROADMAP §2.                                                                             |
| P4 | **Saves consumers hand-rolling backoff.** Four known consumers exist; a shared, tested loop beats four ad-hoc `time.Sleep` ladders.                                                                                                                           | —                                                                                       |

P1 deserves emphasis: the usual "another dependency" objection **does not
apply here**. The cost is not dependency weight. The cost is everything below.

## 3. CONTRA — why the answer is still no

### 3.1 go-sse has no retryable operation (decisive)

Every candidate site was examined. All four fail on technical grounds, not on
taste:

| Candidate site                                | Retryable? | Why not                                                                                                                                                                                                                            |
| --------------------------------------------- | ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Stream.Send` / `WriteEvent` (`stream.go:99`) | **No**     | A failed write to `http.ResponseWriter` means the TCP connection is dead or **partially written**. Retrying re-emits bytes already on the wire, corrupting SSE frames. There is no way to know how much of the frame landed.       |
| `Replay` store query (`replay.go:41`)         | **No**     | `EventStore` is a consumer-supplied interface. Whether a store's failure is worth retrying is the store implementer's policy, not the transport's. Baking in a retry would override the consumer's own DB retry layer.             |
| `Shutdown` drain poll (`fanout.go`)           | **No**     | The drain is a _condition-wait_ ("are all buffers empty yet?"), not a failing operation. Exponential backoff would **increase** p50 shutdown latency versus the current 1 ms poll — strictly worse for the common fast-drain case. |
| `Broadcast` channel send (`fanout.go`)        | **No**     | Concurrency invariant #2 (AGENTS.md) forbids blocking under the read lock. Drop-on-full is documented, intentional, and load-bearing; a retry loop is exactly the head-of-line blocking that invariant prevents.                   |
| Client reconnection                           | **N/A**    | go-sse is server-only. ROADMAP §2 explicitly defers the client `Dial` helper "until a concrete client consumer exists."                                                                                                            |

The only place retry belongs is the one component the project has
**deliberately decided not to build yet**.

### 3.2 The default predicate would do the wrong thing (severity: high)

`errorfamily.IsRetryable` returns true for **exactly** the `Transient` family:

```go
func (f Family) IsRetryable() bool { return f == Transient }
```

go-sse classifies its **connection-death** errors as `Transient`:
`sse.write_failed`, `sse.send_failed`, `sse.replay_failed`.

So the obvious-looking `retry.Do(ctx, retry.DefaultConfig(), send)` would, with
the **default** config, retry a broken pipe three times with backoff. In
go-sse, `Transient` means _"drop this connection and let the browser
reconnect"_ — it does **not** mean _"sleep and try the same dead socket
again."_ Adopting `go-retry` would put a footgun one autocomplete away.

### 3.3 Backoff inside `Stream.Send` would stall the heartbeat

`Send` holds `s.mu` for its whole duration, and that mutex also serializes
`Heartbeat` (invariant #1). A backoff sleep inside the locked region would
block the heartbeat goroutine for the entire retry window — turning a
transient blip into a proxy-killed connection. Retrying _outside_ the lock is
the same as reconnecting, which the browser already does for free.

### 3.4 Naming collision inside one package

`sse` already owns the word "retry", with a completely different meaning:

- `Event.Retry` / `WriteRetry` — the SSE wire field; a hint to the **browser**,
  in milliseconds, about how long to wait before reconnecting.
- `retry.Config` — a **server-side** loop that re-invokes a function.

Two unrelated concepts sharing one word in one namespace is precisely the
"name smuggling" failure mode. Any reader seeing `retry` in this package would
have to disambiguate every time.

### 3.5 Upstream split brain: `RetryPolicy` vs `Config`

`go-error-family` — already a go-sse dependency — **ships its own retry
parameter type** (`retry.go`):

```go
type RetryPolicy struct {
    MaxAttempts int
    MinDelay    time.Duration
    MaxDelay    time.Duration
}
func (f Family) RetryPolicy() RetryPolicy
```

`go-retry` depends on `go-error-family`, uses its `IsRetryable`, and then
**ignores `RetryPolicy` entirely**, defining a competing `Config` with
overlapping-but-differently-named fields (`InitialDelay` vs `MinDelay`).
Adopting `go-retry` would import both types into go-sse's graph. This should
be reconciled upstream before any downstream adoption.

### 3.6 Three reachable panics in v0.1.0 (severity: high)

All three were reproduced with runnable programs, not inferred. Root cause is
`retry.go:137-139`:

```go
delay += time.Duration(rand.Int64N(int64(delay) / 2))
```

`rand.Int64N` panics on a non-positive argument.

| #  | Trigger                                                                                                                                                                                                                           | Reachable via                                                 |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| B1 | **`MaxDelay` is never validated.** `Config.Validate()` checks `MaxAttempts`, `InitialDelay`, and `Multiplier` — but not `MaxDelay`. A hand-built `Config` with `MaxDelay` unset gives `min(delay, 0) == 0` → `Int64N(0)` → panic. | `retry.Do` with any struct literal that omits `MaxDelay`.     |
| B2 | **Sub-2 ns delays.** Any `delay < 2ns` makes `int64(delay)/2 == 0` → panic. `InitialDelay: 1` passes `Validate()`.                                                                                                                | `retry.Do`, `Backoff`, `ComputeDelay`.                        |
| B3 | **`math.Pow` overflow.** `float64 → time.Duration` for an out-of-range value yields `INT64_MIN` on amd64; `min(negative, MaxDelay)` keeps the negative → panic. **Plain `DefaultConfig()` panics at attempt 38.**                 | `retry.Do` with `MaxAttempts >= 38`, or a large `Multiplier`. |

Observed output:

```text
Config{MaxAttempts:3,InitialDelay:10ms,Mult:2}   PANIC: invalid argument to Int64N
overflow via Multiplier=10, MaxAttempts=15       PANIC: invalid argument to Int64N
DefaultConfig() overflows at attempt=38
```

This is not disqualifying on its own — it is a young library and the bugs are
shallow. But it does mean adoption today would import a library whose
100%-statement-coverage claim does not translate into numeric robustness.
`go-retry`'s own `TODO_LIST.md` T1 already flags "fuzz `ComputeDelay` for
numeric edges" as open work; these are the bugs that item predicts.

A verified fix (clamp non-positive delays, saturate the jitter addition,
handle `Inf`/`NaN`, treat unset `MaxDelay` as `InitialDelay`) was validated
against an 84,000-case matrix of `initial × maxDelay × multiplier × attempt`
and is recorded in `go-retry`'s `TODO_LIST.md`.

### 3.7 Scope

go-sse's charter (AGENTS.md) is "two small dependencies; no domain opinions."
Retry policy is a domain opinion. The library's stated non-goals already
reject `Broadcaster.ServeSSE` on exactly this reasoning: a convenience wrapper
that bakes in timing decisions belongs in the consumer.

## 4. Decision

**Do not adopt.** Not because `go-retry` is bad or heavy — P1 shows the
dependency cost is genuinely near zero — but because **go-sse contains no
operation that should be retried**. Adding it would provide no call site,
introduce a default-predicate footgun (§3.2), collide with existing
terminology (§3.4), and import an unresolved upstream type overlap (§3.5).

### Re-open triggers

Revisit when **any** of these fires:

1. A client `Dial` / reconnect helper is being written (ROADMAP §2). This is
   the expected trigger and the genuine use case.
2. A batteries-included `EventStore` with a network backend (e.g. Redis) ships
   in-tree — then go-sse owns the failing I/O and owns the retry policy with it.
3. Upstream reconciles `errorfamily.RetryPolicy` with `retry.Config`, and the
   panics in §3.6 are fixed and released.

Consumers who want retry around their own `EventStore` or handler today should
depend on `go-retry` **directly**. That composition works fine and needs
nothing from go-sse.
