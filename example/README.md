# go-sse examples

Three runnable examples for the [go-sse](https://github.com/larsartmann/go-sse)
library, progressing from a raw terminal demo to two browser UIs that contrast
the two most popular ways to consume SSE in a Go backend.

| Example                  | Port    | What it shows                                                                                            | Audience                              |
| ------------------------ | ------- | -------------------------------------------------------------------------------------------------------- | ------------------------------------- |
| [`server.go`](server.go) | `:8080` | Bare wire format — subscribe and broadcast with `curl`. No browser.                                      | Learning the SSE wire format          |
| [`datastar/`](datastar/) | `:8765` | Live activity feed: fan-out, filtering, replay, heartbeat — via DataStar reactive signals + keyed lines. | Rich, signal-driven UIs               |
| [`htmx/`](htmx/)         | `:8766` | HTML fragment swapping via HTMX + its SSE extension. The server streams HTML; HTMX swaps it in.          | Hypermedia / server-rendered HTML UIs |

The two browser examples take **different scopes** to highlight each tool's
strengths: `datastar/` is a full go-sse feature showcase (a live activity feed
with fan-out, filtering, replay, and heartbeat), while `htmx/` is a focused
progress-bar demo. What's comparable is the _mechanism_ each uses to consume
SSE — summarized below and explained in detail in [DataStar vs HTMX](#datastar-vs-htmx).

---

## Run any example

```bash
go run example/server.go      # raw terminal demo   → http://localhost:8080
go run example/datastar/      # DataStar UI         → http://localhost:8765
go run example/htmx/          # HTMX UI             → http://localhost:8766
```

> Each example is an independent `package main`. Run the package
> (`example/datastar/`), not a single file, because the page is rendered by
> generated templ code (`index_templ.go`). The JS bundles and CSS are embedded
> via `go:embed` — no CDN, no internet connection required at runtime.

---

## 1. Raw server (`server.go`)

A minimal fan-out server. Open a second terminal and broadcast a message;

every connected client receives it instantly.

### Subscribe (terminal 1)

```bash
curl -N http://localhost:8080/events
```

```
event: connected
data: connected
id: 0

: heartbeat
```

### Broadcast (terminal 2)

```bash
curl -X POST http://localhost:8080/broadcast?msg=hello
```

Terminal 1 immediately shows:

```
event: message
data: hello
id: 1770000000000000000
```

`EventSource` handles reconnection automatically and sends the `Last-Event-ID`
header on reconnect, enabling replay of missed events.

---

## 2. DataStar example (`datastar/`)

A **live activity feed** that showcases the full go-sse feature set:

- **`Broadcaster`** — open multiple tabs; every client receives the same events
  simultaneously (real-time fan-out).
- **`SubscribeFilter`** — toggle "Alerts only" via `?filter=alerts`; each
  subscriber's predicate filters events server-side.
- **`EventStore` + `Replay`** — disconnect and reconnect; missed events replay
  from the in-memory ring buffer using `Last-Event-ID`.
- **`Heartbeat`** — comment-frame pings keep the connection alive through
  proxies.
- **Reactive signals** — `datastar-patch-signals` drives the live client count
  (`$subscriberCount`) and replay banner (`$replayed`); `datastar-patch-elements`
  appends entries to the `#feed`.

Built with the go-sse DataStar helpers: [`KeyedLines`](../event.go),
[`SendKeyed`](../stream.go), and [`SendLines`](../stream.go). The page is a
type-safe [templ](https://templ.guide) template; the DataStar JS bundle is
embedded.

```bash
go run example/datastar/    # → http://localhost:8765
```

---

## 3. HTMX example (`htmx/`)

The server emits a single named event, `progress`, whose `data:` payload is an
**HTML fragment**. HTMX's [SSE extension](https://htmx.org/extensions/sse/)
swaps each fragment into the `#progress-area` target (`innerHTML`) as it
arrives. The Restart button fetches a fresh `#sse-container` fragment and swaps
it in (`outerHTML`), which tears down the old `EventSource` and opens a new one
— no JavaScript required.

Built with plain `stream.Send(sse.Event{...})`; HTMX speaks vanilla SSE, so no
HTMX-specific helpers are needed. htmx 2.0.4 + htmx-ext-sse 2.2.4 are vendored.

```bash
go run example/htmx/        # → http://localhost:8766
```

---

## DataStar vs HTMX

The two examples demonstrate different scopes (DataStar is a full feature
showcase; HTMX is a focused demo), but what matters for comparison is the
_mechanism_: what the server sends over the wire and how the client applies
updates. The trade-offs below flow from that mechanism, not the specific demo.

### At a glance

| Aspect                    | DataStar `datastar/`                                                               | HTMX `htmx/`                                                |
| ------------------------- | ---------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| **Mental model**          | Reactive signals + surgical DOM patches                                            | Hypermedia: swap HTML fragments into targets                |
| **Server sends**          | Keyed-line protocol (`selector …`, `mode …`, `signals {…}`)                        | Raw HTML fragments                                          |
| **go-sse API used**       | `Broadcaster`, `SubscribeFilter`, `EventStore`/`Replay`, `Heartbeat`, `KeyedLines` | `Send(Event{...})` (vanilla SSE; no special helpers)        |
| **Client bundle**         | `datastar.js` (~56 KB, 1 file)                                                     | `htmx.min.js` + `sse.min.js` (~54 KB, 2 files)              |
| **Payload per event**     | Small — only the changed signal / element                                          | Larger — the whole fragment is re-rendered each event       |
| **Update granularity**    | Surgical — updates one value on a **stable** element                               | Coarse — `innerHTML` swap **recreates** the children        |
| **In-place vs re-render** | Patches a stable node (CSS transitions & focus persist)                            | Replaces children (transitions reset; use `morph` ext)      |
| **Restart the stream**    | `data-on:click="@get('/events')"` — one attribute                                  | Swap the `sse-connect` container (`outerHTML`) — a fragment |
| **Reconnect / replay**    | Native `EventSource` → `Last-Event-ID` → go-sse replay                             | Native `EventSource` → `Last-Event-ID` → go-sse replay      |
| **Ecosystem / maturity**  | Newer, custom attribute DSL, growing                                               | Mature, huge ecosystem, standard HTML attributes            |
| **Learning curve**        | Reactivity + DataStar expression DSL                                               | Hypermedia fundamentals (HTML in, HTML out)                 |

### Pros & cons

#### DataStar

**Pros**

- **Fine-grained updates.** Patch a single signal (`signals {"count": 3}`)
  or append one element without resending surrounding markup. Smaller payloads,
  lower bandwidth at scale.
- **Reactive by default.** Bind a signal once (`data-text="$subscriberCount"`)
  and every server-side update flows into the DOM automatically — no manual
  targeting or fragment endpoints.
- **Stable elements.** Updates patch a node in place rather than replacing it,
  so CSS transitions and input focus persist across updates.
- **First-class in go-sse.** `KeyedLines` / `SendKeyed` / `SendLines` /
  `WriteKeyedLines` exist specifically to build the keyed-line wire format
  correctly (multi-line splitting, CRLF handling) with zero boilerplate.

**Cons**

- **Proprietary protocol.** The keyed-line format (`selector`, `mode`,
  `elements`, `signals`) is DataStar-specific; clients must run DataStar.
- **Newer ecosystem.** Smaller community, fewer integrations, evolving docs.
- **DSL to learn.** `data-signals`, `data-on:click`, `@get()`, `$var`
  expressions are a custom mental model on top of HTML.
- **Heavier conceptually** for teams that only need "replace this div".

#### HTMX

**Pros**

- **Universal.** The server sends ordinary HTML — anything your templ / `html/template`
  / `text/template` already renders works as an SSE fragment. No special
  protocol or wire-format helpers.
- **Mature & familiar.** Standard attributes (`hx-get`, `hx-swap`,
  `sse-swap`), a large ecosystem, and the hypermedia model is easy to reason
  about: HTML in, HTML out.
- **Composable with the rest of HTMX.** The same fragment endpoint serves both
  SSE swaps and regular HTMX AJAX requests; SSE is just another trigger.
- **Shallow go-sse integration** is a feature here — no library-specific
  coupling; plain `Event` structs.

**Cons**

- **Coarse granularity.** `innerHTML` swap recreates the target's children on
  every event. Unchanged surrounding HTML is resent and re-rendered, and CSS
  transitions on replaced elements reset. Smoother updates need the extra
  `idiomorph` extension (`hx-swap="morph"`).
- **Larger payloads.** Each event carries the entire fragment, not just the
  delta — more bytes per tick, which adds up for high-frequency streams.
- **Restart is more ceremony.** Re-establishing an SSE stream means replacing
  the `sse-connect` element (a fragment swap), not a single attribute.
- **Two client files.** Core htmx + the SSE extension must both be loaded.

### When to choose which

| Pick DataStar if…                                              | Pick HTMX if…                                                         |
| -------------------------------------------------------------- | --------------------------------------------------------------------- |
| You need fine-grained, low-bandwidth reactive updates          | You already server-render HTML and want to stream it as-is            |
| Smooth in-place animations matter (progress bars, live gauges) | Your team knows hypermedia and wants the simplest possible model      |
| You're building a signal-driven, app-like UI                   | You want one fragment endpoint that also serves regular HTMX requests |
| You're happy to adopt the DataStar attribute DSL               | You want to stay on standard HTML attributes and a mature ecosystem   |

### The bottom line

**DataStar** optimizes for _precision_ — small, surgical updates to reactive
state and individual elements, at the cost of a custom protocol and DSL.
**HTMX** optimizes for _simplicity and universality_ — stream any HTML and let
HTMX swap it, at the cost of coarser updates and larger payloads. go-sse
supports both first-class: purpose-built helpers for DataStar's keyed lines,
and plain `Event` sending for HTMX's fragment streams. Both reuse the same
reconnection, replay, fan-out, and heartbeat infrastructure.
