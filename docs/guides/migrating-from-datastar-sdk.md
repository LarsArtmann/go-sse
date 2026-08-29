# Migrating from the DataStar Go SDK to go-sse

The official DataStar Go SDK (`github.com/starfederation/datastar-go/datastar`)
provides typed builders for every DataStar event type. go-sse takes a different
approach: general-purpose SSE primitives that compose into the DataStar wire
format without coupling your server to a specific frontend framework.

This guide maps every common DataStar SDK operation to its go-sse equivalent.

## Why migrate?

| Concern                | DataStar Go SDK                                               | go-sse                                            |
| ---------------------- | ------------------------------------------------------------- | ------------------------------------------------- |
| Scope                  | DataStar-specific typed builders                              | General SSE transport                             |
| Dependencies           | `bytebufferpool`, `http.ResponseController`, DataStar runtime | Two small same-author modules                     |
| Wire format            | Hidden behind typed API                                       | Explicit — you see every `data:` line             |
| Framework coupling     | Tied to DataStar event names                                  | `KeyedLines` is general; DataStar is one consumer |
| Replay / reconnection  | Not provided                                                  | `EventStore` + `Replay` + `LastEventID`           |
| Heartbeat / keep-alive | Not provided                                                  | `Stream.Heartbeat` (proxy survival)               |

## Quick reference

### Setup

**DataStar SDK:**

```go
import "github.com/starfederation/datastar-go/datastar"

func handler(w http.ResponseWriter, r *http.Request) {
    sse := datastar.NewSSE(w, r)
    defer sse.Close()
    // ...
}
```

**go-sse:**

```go
import "github.com/larsartmann/go-sse"

func handler(w http.ResponseWriter, r *http.Request) {
    stream := sse.NewStream(w, r)
    defer func() { _ = stream.Close() }()
    // ...
}
```

### Patch elements

**DataStar SDK:**

```go
sse.PatchElements(`<div id="feed">Hello</div>`,
    datastar.WithMode(datastar.ElementPatchModeInner),
    datastar.WithSelector("#feed"),
)
```

**go-sse:**

```go
stream.SendLines("datastar-patch-elements",
    "selector #feed",
    "mode inner",
    sse.KeyedLines("elements", `<div id="feed">Hello</div>`),
)
```

### Multi-line HTML fragments

**DataStar SDK:**

```go
sse.PatchElements(`<div id="feed">
  <article>Item 1</article>
  <article>Item 2</article>
</div>`, datastar.WithSelector("#feed"))
```

**go-sse:**

```go
html := `<div id="feed">
  <article>Item 1</article>
  <article>Item 2</article>
</div>`

stream.SendLines("datastar-patch-elements",
    "selector #feed",
    sse.KeyedLines("elements", html),
)
```

`KeyedLines` prefixes every line with `elements` and `WriteEvent` splits the
result into individual `data:` lines automatically.

### Patch signals (from struct/map)

**DataStar SDK:**

```go
sse.MarshalAndPatchSignals(map[string]any{
    "progress": 75,
    "status":   "loading",
})
```

**go-sse:**

```go
payload, err := json.Marshal(map[string]any{
    "progress": 75,
    "status":   "loading",
})
if err != nil { return err }

stream.SendKeyed("datastar-patch-signals", "signals", string(payload))
```

### Patch signals (from raw JSON)

**DataStar SDK:**

```go
sse.PatchSignals([]byte(`{"progress": 75}`))
```

**go-sse:**

```go
stream.SendKeyed("datastar-patch-signals", "signals", `{"progress": 75}`)
```

### Remove element

**DataStar SDK:**

```go
sse.RemoveElement("#feed")
```

**go-sse:**

```go
stream.SendLines("datastar-patch-elements",
    "selector #feed",
    "mode remove",
)
```

### Execute script

**DataStar SDK:**

```go
sse.ExecuteScript(`console.log("hello")`)
```

**go-sse:**

```go
stream.SendLines("datastar-execute-script",
    sse.KeyedLines("script", `console.log("hello")`),
)
```

### Set retry duration

**DataStar SDK:**

```go
sse.PatchElements(html, datastar.WithRetryDuration(2*time.Second))
```

**go-sse:**

```go
stream.Send(sse.Event{
    Event: "datastar-patch-elements",
    Data:  sse.KeyedLines("elements", html),
    Retry: 2000, // milliseconds
})
```

Or set it globally with `WriteRetry`:

```go
sse.WriteRetry(w, 2000) // affects all subsequent events from the browser
```

### Read client signals (from request)

**DataStar SDK:**

```go
var signals struct {
    Name  string `json:"name"`
    Count int    `json:"count"`
}
datastar.ReadSignals(r, &signals)
```

**go-sse:**

go-sse does not provide a signals reader (it is a transport library, not a
DataStar framework). Parse the request body directly:

```go
var signals struct {
    Name  string `json:"name"`
    Count int    `json:"count"`
}
json.NewDecoder(r.Body).Decode(&signals)
```

DataStar sends signals as JSON in the request body (POST/PUT/PATCH) or as the
`datastar` query parameter (GET).

## What go-sse gives you that the DataStar SDK does not

- **Fan-out**: `Broadcaster[T]` — non-blocking broadcast to N subscribers with
  O(1) unsubscribe and drop-on-full policy.
- **Reconnection replay**: `EventStore` interface + `Replay` function +
  `Stream.LastEventID()` — recover missed events after a connection drop.
- **Heartbeat**: `Stream.Heartbeat` — SSE comment frames at an interval to
  survive reverse-proxy idle timeouts (Nginx, Cloudflare, AWS ALB).
- **Branded `EventID`**: phantom-type IDs that prevent cross-assignment with
  other string-typed IDs.
- **Allocation-minimized serialization**: `WriteEvent` uses direct byte appends,
  not `fmt.Fprintf`, on the hot path.

## What the DataStar SDK gives you that go-sse does not

- **Typed builders**: `WithMode`, `WithSelector`, `WithViewTransitions`, etc.
  — go-sse requires manual data-line construction.
- **Templ integration**: `PatchElementTempl` renders templ components directly.
- **GoStar integration**: `PatchElementGostar` renders GoStar elements.
- **Convenience methods**: `Redirect`, `ConsoleLog`, `DispatchCustomEvent`,
  `ReplaceURL`, `Prefetch` — all sugar over `datastar-execute-script`.

If you need these, you can use the DataStar SDK alongside go-sse's Broadcaster
and Replay — they operate at different layers.
