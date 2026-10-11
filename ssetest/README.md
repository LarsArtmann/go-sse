# ssetest

Deep-testing helpers for [go-sse](https://github.com/LarsArtmann/go-sse) consumers.

Testing an SSE handler should not require hand-rolling a wire-format parser or
squinting at `data:` lines. `ssetest` gives you one-liners that spin up a real
HTTP server, drive your handler, and hand back typed, assertable events.

```bash
go get github.com/larsartmann/go-sse/ssetest
```

The WHATWG-conformant parsing core lives in its own zero-dependency module,
[`sseparse`](../sseparse/). ssetest re-exports its entire API (events, readers,
assertions), so handler tests need one import — but consumers that parse
non-HTTP streams (unix sockets, pipes) or want the conformance corpus as a
spec oracle can `go get` sseparse alone, with nothing go-sse-shaped in their
module graph.

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

## Reading from any io.Reader

Not every SSE stream comes from an `http.Handler` — and not every consumer is
an HTTP client. The reader API parses the wire format directly:

```go
// Parse a complete stream (reads until EOF — e.g. a captured response body):
events := ssetest.ReadEvents(r)

// Read exactly N events from a live stream, then stop:
events := ssetest.ReadNEvents(r, 3)

// Keep one scanner across many reads — no buffered data lost between calls.
// The pattern for interleaving reads with actions (POST, mutate, read next):
reader := ssetest.NewStreamReader(r)
evt := ssetest.MustReadNextEvent(t, reader)
```

A single SSE line must fit within `ssetest.DefaultMaxLineBytes` (1 MiB,
terminator included); over the cap the read fails with an error wrapping
`bufio.ErrTooLong` (`errors.Is`-matchable). Mirror your own parser's bound
with `ssetest.WithMaxLineBytes(n)` — accepted by `ReadEvents`, `ReadNEvents`,
and `NewStreamReader`.

This API is ideal for unix-socket daemons and other non-HTTP transports; in
that case import [`sseparse`](../sseparse/) directly and skip the HTTP
helpers (and the go-sse require) entirely.

## Collecting events over HTTP

| Helper                                                | Use when                                               |
| ----------------------------------------------------- | ------------------------------------------------------ |
| `Collect(t, handler, opts...)`                        | Handler sends its events and returns (GET)             |
| `CollectPost(t, handler, jsonBody, opts...)`          | POST with a JSON body                                  |
| `CollectWithRequest(t, h, method, body, ct, opts...)` | Any method/body/content-type                           |
| `CollectN(t, handler, count, opts...)`                | Streaming handler; reads exactly N events, then closes |
| `CollectWithTimeout(t, handler, timeout, opts...)`    | Time-bounded read; returns the events that arrived     |

Per the SSE specification, frames without a `data:` line (comments, heartbeats,
id/retry-only frames) never surface as events — the parser matches browser
behavior, so heartbeated streams test cleanly.

## Spec conformance

The parser implements the WHATWG HTML Living Standard
[§ 9.2.6](https://html.spec.whatwg.org/multipage/server-sent-events.html#parsing-an-event-stream)
and is pinned by the official browser test suite: the Web Platform Tests
`eventsource/format-*` corpus, the spec's own example streams, and Chromium's
`event_source_parser_test.cc` unit cases — exported as data
(`sseparse.Corpus()`), executed as Go tests
(`sseparse/wpt_format_corpus_test.go`), re-run through 1–4096 byte chunked
readers (`sseparse/chunk_boundary_test.go`) to prove TCP-chunking
independence, and closed against the writer via round-trip property tests
(`roundtrip_test.go`). In short, `ReadEvents` parses exactly what a browser's
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

Writing your own SSE parser? The corpus is a ready-made spec oracle — see
[../sseparse/](../sseparse/README.md#the-conformance-corpus-as-a-spec-oracle).

## Request options

Every `Collect*` helper accepts options:

```go
// Target a route on a mux (query strings allowed):
ssetest.Collect(t, mux, ssetest.WithPath("/events?filter=alerts"))

// Simulate a reconnecting browser for replay testing:
events := ssetest.Collect(t, handler, ssetest.WithLastEventID(sse.NewEventID("42")))

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
