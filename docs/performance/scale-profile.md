# Scale Profile — go-sse Broadcaster

> Memory and latency characterization of the fan-out hot path at 100 / 1k / 10k
> subscribers. Answers: is the default 64-event buffer and non-blocking drop
> policy appropriate, or do they need to change?

**Last measured:** 2026-08-03 · **Hardware:** AMD Ryzen AI MAX+ 395 (Linux/amd64) · **Go:** 1.26 (`GOEXPERIMENT=jsonv2`)

## Reproduce

```bash
GOWORK=off GOEXPERIMENT=jsonv2 go test ./... -run='^$' \
  -bench='BenchmarkBroadcasterFanOut|BenchmarkSubscribeFilter_PredicateOverhead|BenchmarkMemoryPerSubscriber' \
  -benchmem -benchtime=200ms -count=1
```

## Latency — `Broadcast` (unfiltered, draining consumers)

| Subscribers | Total per broadcast | Per-subscriber cost |
| ----------- | ------------------- | ------------------- |
| 1           | 77 ns               | 77 ns               |
| 10          | 317 ns              | 32 ns               |
| 100         | 5.9 µs              | 59 ns               |
| 1 000       | 212 µs              | 212 ns              |
| 10 000      | 9.1 ms              | 913 ns              |

**Allocations on the hot path: 0 B/op, 0 allocs/op at every scale.** The broadcast
loop allocates nothing — sends use a non-blocking `select` into pre-allocated
channels. GC pressure from fan-out itself is zero.

## Latency — predicate overhead (filtered vs unfiltered)

| Subscribers | Unfiltered | Filtered | Factor |
| ----------- | ---------- | -------- | ------ |
| 1           | 64 ns      | 103 ns   | 1.6×   |
| 100         | 4.2 µs     | 10 µs    | 2.4×   |
| 1 000       | 260 µs     | 217 µs   | ~1×    |

The filtered path carries the `safePredCall` defer/recover overhead plus the
predicate function call. At 1 subscriber this is ~40 ns absolute — negligible.
At 1000 subscribers cache effects dominate and the two converge. The recovery
wrapper is never a bottleneck.

## Memory per subscriber (default 64-event buffer)

| Subscribers | Total heap | Per subscriber |
| ----------- | ---------- | -------------- |
| 100         | 400 KiB    | ~4.0 KiB       |
| 1 000       | 4.2 MiB    | ~4.3 KiB       |
| 10 000      | 41.7 MiB   | ~4.3 KiB       |

**The buffer dominates.** `sse.Event` is 56 bytes (3 strings + 1 uint); a
64-deep channel holds 64 × 56 = 3.5 KiB of buffer alone. The remaining ~0.8 KiB
is the channel header, the `subscriber[T]` struct, and the map entry.

## Conclusion: no change needed

1. **Buffer size (64) is well-calibrated.** ~4 KiB per subscriber means 10 000
   concurrent connections cost ~42 MiB — a comfortable floor for any server that
   already holds 10 000 open HTTP connections. Deployments that need more headroom
   (fewer drops under burst) or less memory (more drops) can tune via
   `WithBufferSize` — that is exactly the escape hatch it was designed to be.

2. **Non-blocking drop policy is correct.** A single slow consumer can never stall
   the fan-out. At 10 000 subscribers a broadcast completes in ~9 ms; if even one
   subscriber's buffer were full and the send blocked, every other subscriber
   would wait on that one. The drop policy keeps the tail latency bounded.

3. **No backpressure mechanism is warranted.** Adding one (e.g. blocking sends,
   per-subscriber flow control) would reintroduce head-of-line blocking and
   violate invariant 2. Consumers needing guaranteed delivery already have the
   `EventStore` + `Replay` reconnection path.

## When to revisit

- If real-world deployments routinely exceed 50 000 concurrent subscribers,
  re-measure and consider whether the default buffer should drop to 32.
- If a workload shows frequent drops under burst (observable via a future
  drop-counter metric), the first lever is `WithBufferSize`, not a policy change.
