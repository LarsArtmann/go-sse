# ssetest

Deep-testing helpers for [go-sse](https://github.com/LarsArtmann/go-sse) consumers.

Testing an SSE handler should not require hand-rolling a wire-format parser or
squinting at `data:` lines. `ssetest` gives you one-liners that spin up a real
HTTP server, drive your handler, and hand back typed, assertable events.

```bash
go get github.com/larsartmann/go-sse/ssetest
```

## Quick start

```go
import (
    "net/http"
    "testing"

    "github.com/larsartmann/go-sse"
    "github.com/larsartmann/go-sse/ssetest"
)

func TestFeedHandler(t *testing.T) {
    t.Parallel()

    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        stream := sse.NewStream(w, r)
        defer func() { _ = stream.Close() }()

        _ = stream.Send(sse.Event{Event: "feed", Data: "hello"})
    })

    events := ssetest.Collect(t, handler)
    ssetest.RequireEventCount(t, events, 1)
    ssetest.RequireEventType(t, events[0], "feed")
    ssetest.RequireData(t, events[0], "hello")
}
```

## Collecting events

| Helper                                                | Use when                                               |
| ----------------------------------------------------- | ------------------------------------------------------ |
| `Collect(t, handler, opts...)`                        | Handler sends its events and returns (GET)             |
| `CollectPost(t, handler, jsonBody, opts...)`          | POST with a JSON body                                  |
| `CollectWithRequest(t, h, method, body, ct, opts...)` | Any method/body/content-type                           |
| `CollectN(t, handler, count, opts...)`                | Streaming handler; reads exactly N events, then closes |
| `CollectWithTimeout(t, handler, timeout, opts...)`    | Time-bounded read; returns the events that arrived     |
| `ReadEvents(r)` / `ReadNEvents(r, n)`                 | Parse SSE from any `io.Reader` yourself                |

Per the SSE specification, frames without a `data:` line (comments, heartbeats,
id/retry-only frames) never surface as events — the parser matches browser
behavior, so heartbeated streams test cleanly.

## Spec conformance

The parser implements the WHATWG HTML Living Standard
[§ 9.2.6](https://html.spec.whatwg.org/multipage/server-sent-events.html#parsing-an-event-stream)
and is pinned by the official browser test suite: the Web Platform Tests
`eventsource/format-*` corpus, the spec's own example streams, and Chromium's
`event_source_parser_test.cc` unit cases are transcribed as Go tests
(`wpt_format_corpus_test.go`), re-run through 1–4096 byte chunked readers to
prove TCP-chunking independence, and closed against the writer via round-trip
property tests. In short, `ReadEvents` parses exactly what a browser's
`EventSource` parses:

- Lines end with CR, LF, or CRLF; a blank line dispatches a frame.
- Exactly one leading UTF-8 BOM is stripped; a mid-stream BOM is data.
- Field names are case-sensitive; unknown fields and `:` comments are ignored;
  one space after the colon is stripped (never a tab).
- A `data:` line with an empty value still dispatches an event (empty payload);
  frames with no `data:` line never dispatch.
- `id:` is sticky connection state: events report the most recent `id:` value
  (empty `id:` resets it; NUL-containing `id:` is ignored).
- `retry:` is sticky connection state too: all-ASCII-digit values (leading
  zeros allowed) update it even in frames that never dispatch; invalid values
  never reset it.
- An incomplete final frame at EOF is discarded, per the spec's
  "any pending data must be discarded" rule.

## Request options

Every `Collect*` helper accepts options:

```go
// Target a route on a mux (query strings allowed):
ssetest.Collect(t, mux, ssetest.WithPath("/events?filter=alerts"))

// Simulate a reconnecting browser for replay testing:
events := ssetest.Collect(t, handler, ssetest.WithLastEventID("42"))

// Any custom header:
ssetest.Collect(t, handler, ssetest.WithHeader("X-Trace", "abc"))
```

## Assertions

```go
ssetest.RequireEventCount(t, events, 2)
ssetest.RequireEventType(t, events[0], "feed")
ssetest.RequireData(t, events[0], "hello")
ssetest.RequireDataContains(t, events[0], "hello")
ssetest.RequireEventID(t, events[0], "42")
ssetest.RequireRetry(t, events[0], 3000)
```

All helpers accept [`testing.TB`](https://pkg.go.dev/testing#TB), so they work
with `*testing.T`, `*testing.B`, and Ginkgo's `GinkgoT()`.

## Finding events without index math

```go
evt, ok := ssetest.FindByType(events, "feed")
feeds := ssetest.FilterByType(events, "feed")
```

## Debugging failures

```go
t.Fatalf("unexpected events:\n%s", ssetest.EventsString(events))
```

The package is a separate Go module (it depends on `testing`), so it never
leaks into production builds of go-sse consumers. See
[pkg.go.dev](https://pkg.go.dev/github.com/larsartmann/go-sse/ssetest) for the
complete API.
