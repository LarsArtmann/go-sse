# EventStore Patterns: Retention and GC for Replay Stores

The [`EventStore`][store] interface is deliberately unopinionated:

```go
type EventStore interface {
    EventsAfter(id EventID) ([]Event, error)
}
```

That minimalism puts one dangerous decision in your hands: **how long events
live**. A store that never forgets is a memory leak that survives deploys —
every broadcast grows it, nothing shrinks it, and each reconnecting client
replays a longer and longer tail. This guide shows the retention patterns that
keep replay bounded, and the semantics you must pick when events are gone.

[store]: https://pkg.go.dev/github.com/larsartmann/go-sse#EventStore

## What Replay actually asks of a store

`Replay(stream, store, lastEventID)` calls `EventsAfter(id)` **once** and
writes everything it returns. Three properties follow:

1. **The return slice bounds the worst-case reconnect.** A client that
   reconnects after a week must not replay a week of events — unless your
   domain says it should. Retention policy _is_ your reconnect budget.
2. **An empty `EventID` means "from the beginning."** On first connect the
   browser sends no `Last-Event-ID`; `Replay` then receives the zero ID and
   your store decides what "the beginning" is. For a ring buffer, that is
   "whatever is still retained."
3. **Order must be append order.** The browser dispatches what it receives;
   gaps and reordering surface as UI bugs, not errors.

## Pattern 1: Bounded ring buffer (the default choice)

Keep the last N events. On append, evict the oldest. This is what
`example/datastar/store.go` (`memStore`) does:

```go
type memStore struct {
    mu     sync.Mutex
    events []sse.Event
    max    int
}

func (m *memStore) Append(evt sse.Event) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.events = append(m.events, evt)
    if len(m.events) > m.max {
        // Copy down instead of re-slicing: the evicted prefix stays
        // referenced (and un-GC-able) by the backing array otherwise.
        m.events = m.events[len(m.events)-m.max:]
    }
}

func (m *memStore) EventsAfter(id sse.EventID) ([]sse.Event, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if id.Get() == "" {
        return slices.Clone(m.events), nil // first connect: replay all retained
    }
    for i, evt := range m.events {
        if evt.ID.Get() == id.Get() {
            return slices.Clone(m.events[i+1:]), nil
        }
    }
    // ID not found — see "Stale IDs" below.
    return slices.Clone(m.events), nil
}
```

Notes that matter in production:

- **Trim on append, not with a background sweeper.** A sweeper races with
  `EventsAfter` snapshots and adds a goroutine to own; trim-at-write keeps
  the invariant local to the lock.
- **`slices.Clone` on every read.** `EventsAfter` returns a slice the caller
  will iterate while your next `Append` may trim — handing out the internal
  backing array is a data race.
- **Size the ring from reconnect reality, not from event volume.** If clients
  reconnect within seconds, a few hundred slots cover them; N should exceed
  the number of events emitted during your clients' _typical_ disconnect
  window (page reload: ~1s; laptop sleep: unbounded — accept the gap).

## Pattern 2: Time-based retention (TTL)

Keep events younger than T. Same shape as the ring buffer, but the trim
condition compares timestamps:

```go
type ttlStore struct {
    mu     sync.Mutex
    events []timestamped // struct { evt sse.Event; at time.Time }
    ttl    time.Duration
}

func (t *ttlStore) Append(evt sse.Event) {
    t.mu.Lock()
    defer t.mu.Unlock()

    now := time.Now()
    t.events = append(t.events, timestamped{evt, now})
    cutoff := now.Add(-t.ttl)
    drop := sort.Search(len(t.events), func(i int) bool {
        return t.events[i].at.After(cutoff)
    })
    t.events = t.events[drop:]
}
```

Use this when the domain answer to "how far back can you reconnect?" is a
duration ("5 minutes of feed"), not a count. It still needs a **count cap as
well** — a traffic spike inside the TTL window can hold more events than
memory allows. Combine both conditions in the trim.

## Pattern 3: Persistent stores (database)

The interface maps onto one indexed query:

```sql
SELECT ... FROM events WHERE id > $1 ORDER BY id ASC LIMIT $2
```

Two rules beyond the SQL:

- **Cap the query (`LIMIT`)** and define what happens at the cap (see
  "Stale IDs"). A client reconnecting from a month ago must not pull a
  month of rows through one HTTP response.
- **Do the query off the write path.** `EventsAfter` runs in the handler
  goroutine; a slow index scan stalls that one connection, not the
  broadcaster — but a connection pool exhausted by replay scans stalls
  everything. Use a dedicated read pool or a lower `StatementTimeout` for
  replay queries.

Persistence pairs naturally with the ring buffer: keep the ring for fast
in-process replay, and treat the table as the source of truth for cold
reconnects (or for audit — a concern replay does not need to serve).

## Stale IDs: pick a semantics and write it down

A client reconnects with a `Last-Event-ID` older than your retention window.
`EventsAfter(id)` finds nothing after (or cannot even find) that ID. Your
store must already have decided:

| Semantics          | `EventsAfter(staleID)` returns    | Good for                                   |
| ------------------ | --------------------------------- | ------------------------------------------ |
| Best-effort replay | everything still retained         | Feeds, dashboards — a gap is acceptable    |
| Snapshot reset     | nothing; you send a "state" event | State-sync protocols (re-fetch, then live) |
| Error              | an error; `Replay` surfaces it    | Contract-critical streams                  |

Best-effort (replay what survived) is the right default; document it in the
store type's doc comment so the choice is discoverable. Whatever you pick,
**never** return events _before_ the requested ID as if they were after it.

## Checklist

- [ ] Store has an explicit retention bound (count, TTL, or both)
- [ ] Trim happens on append under the store's own lock
- [ ] `EventsAfter` returns a copy (or otherwise detaches from the buffer)
- [ ] Empty-ID ("first connect") behavior is chosen and documented
- [ ] Stale-ID behavior is chosen and documented
- [ ] Reconnect worst case = retention window; you can state it in one sentence

## See also

- [Reconnection and Retry](reconnection-and-retry.md) — how `Replay`,
  `Last-Event-ID`, and the browser's reconnect hint cooperate.
- `example/datastar/store.go` — the ring buffer this guide abstracts, in use.
