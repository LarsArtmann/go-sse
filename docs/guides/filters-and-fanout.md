# Filters and Fan-Out Patterns

`Broadcaster` fans every broadcast out to all subscribers;
[`SubscribeFilter`][sf] narrows one subscriber's stream to the events it
cares about. Used well, it replaces a zoo of per-topic channels with one hub
and small predicates. Used wrongly, it stalls every subscriber behind one
slow predicate call. The difference is one rule:

[sf]: https://pkg.go.dev/github.com/larsartmann/go-sse#Broadcaster.SubscribeFilter

> **Predicates run under the fan-out read lock, on the broadcasting
> goroutine.** They must be pure, fast, and non-blocking.

Every recipe below is that rule applied.

## Recipe 1: Event-type filter (the 90% case)

```go
msgs := bc.SubscribeFilter(func(evt sse.Event) bool {
    return evt.Event == "message"
})
```

A field comparison is the ideal predicate: no allocation, no locks, no I/O.
A `nil` predicate means "all events" — identical to `Subscribe`.

## Recipe 2: Per-connection topics from the request

Each handler derives the predicate from *its own* request, so one broadcaster
serves every client's slice of the stream (the `?filter=alerts` pattern from
`example/datastar`):

```go
func eventsHandler(bc *sse.Broadcaster[sse.Event]) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        topics := r.URL.Query()["topic"] // ?topic=alerts&topic=deploys

        ch := bc.SubscribeFilter(func(evt sse.Event) bool {
            for _, want := range topics {
                if evt.Event == want {
                    return true
                }
            }
            return false
        })
        defer bc.Unsubscribe(ch)
        // ... stream loop: forward ch to the Stream
    }
}
```

The closure captures per-connection state — that is the sanctioned way to
keep per-subscriber context. Mutating captured state from inside the
predicate is not: `Broadcast` may run the predicate concurrently with itself
across goroutines (read lock, many broadcasters).

## Recipe 3: Payload inspection — prefix, never parse

```go
// Fine: bounded prefix scan over the payload.
builds := bc.SubscribeFilter(func(evt sse.Event) bool {
    return strings.HasPrefix(evt.Data, "service: checkout")
})

// WRONG: unmarshalling inside the predicate.
// var p payload
// json.Unmarshal([]byte(evt.Data), &p)  // allocation + full parse per
//                                       // subscriber per broadcast, under
//                                       // the read lock
```

If filtering needs the parsed payload, filter on something cheap that
correlates (the event type, an ID prefix, a keyed line), or move the payload
into `Event.Event`/IDs at production time. The predicate sees the raw string
for a reason.

## Recipe 4: What a panic in a predicate does (and why that is safe)

A panicking predicate is recovered (`safePredCall` in `fanout.go`) and
treated as a non-match **for that subscriber only** — the broadcast and the
other subscribers continue. This is a containment net, not a license: a
predicate that panics on some payloads silently drops events. Table-test
your predicates against real payloads.

## Filter-reject vs. buffer-drop: two different losses

Both end with a subscriber not seeing an event; only one of them is
observable:

|                       | Predicate rejects                 | Buffer full (drop)                       |
| --------------------- | --------------------------------- | ---------------------------------------- |
| When                  | during `Broadcast`, before send   | during `Broadcast`, at the channel send  |
| Observable            | no signal                          | `WithOnDrop` / `OnDrop` callback fires   |
| Meaning               | "not for you" (by design)         | "for you, but you were too slow"         |
| Fix                   | none needed                        | drain faster / bigger buffer / replay    |

If silence-on-reject ever matters to your domain, emit a metrics line from
the *producer* side before broadcasting, not from inside the predicate.

## Drop policy interaction: filters do not protect buffers

Filtering reduces how often a subscriber's buffer fills — it does not change
what happens when it does: the event is dropped, not retried, and `onDrop`
fires (per subscriber, per message). A subscriber whose predicate rejects
90% of traffic still loses the 10% it wants if its handler stalls. The
remedies are `WithBufferSize` (deepen the buffer) and replay (see below) —
not filter tuning.

## Pair filters with `ReplayFiltered` for reconnection

On reconnect, `SubscribeFilter` governs the live stream, but the missed-event
tail comes from `Replay` — and an unfiltered replay leaks rejected event
types straight into the client. Keep the two consistent:

```go
pred := func(evt sse.Event) bool { return evt.Event == "message" }

ch := bc.SubscribeFilter(pred) // live events
sse.ReplayFiltered(stream, store, lastID, pred) // missed events, same rule
```

`ReplayFiltered` pushes the predicate into stores that implement
`FilteredEventStore` (efficient) and post-filters plain `EventStore` results
(correct, budget-bounded by the store's retention — see
[EventStore Patterns](eventstore-patterns.md)).

## The anti-pattern checklist

A predicate must not:

- do I/O (file, network, channel, `time.Sleep`)
- acquire locks that any writer path holds (`OnDrop`/`OnDrop`-style setters
  on the broadcaster deadlock re-entrantly)
- allocate unboundedly (`json.Unmarshal`, `fmt.Sprintf` per call)
- mutate shared state without its own synchronization
- call back into the broadcaster (`Subscribe`, `Broadcast`, `Shutdown`)

If you cannot express the filter within these rules, you do not have a
filter problem — you have a routing problem: broadcast a distinct event
type, or run two broadcasters.

## See also

- `fanout.go` — the read-lock contract and `safePredCall` containment.
- `example/datastar/handlers.go` — the `?filter=alerts` recipe, live.
- [EventStore Patterns](eventstore-patterns.md) — retention bounds for the
  replay side of filtering.
