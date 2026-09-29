# sseparse

Zero-dependency, WHATWG-conformant SSE wire-format parser — the parsing core
of [ssetest](../ssetest/), extractable without go-sse in your module graph.

```bash
go get github.com/larsartmann/go-sse/sseparse
```

> Tagged `sseparse/v0.1.0` (2026-09-29): `go get` resolves to the latest
> `sseparse/vX.Y.Z` tag; pass `@v0.1.0` explicitly to pin an exact version.

`sseparse` implements the WHATWG HTML Living Standard
[§ 9.2.6](https://html.spec.whatwg.org/multipage/server-sent-events.html#parsing-an-event-stream)
event-stream interpretation: what a browser's `EventSource` parses, nothing
more and nothing less. It has **no third-party dependencies** — no go-sse, no
go-branded-id, no go-error-family — so unix-socket daemons, protocol
libraries, and other SSE parsers can adopt it (or its conformance corpus)
without those modules landing in a hermetic build's module graph.

## Parse SSE from any io.Reader

```go
events, err := sseparse.ReadEvents(r)
if err != nil {
    t.Fatal(err)
}

sseparse.RequireData(t, events[0], "hello")
```

Non-HTTP streams are the primary use case: unix sockets, pipes, captured
response bodies. `ReadNEvents(r, n)` reads exactly N events from a live
stream; `NewStreamReader(r)` keeps one scanner across many reads (no buffered
data lost between calls):

```go
reader := sseparse.NewStreamReader(conn)
for {
    evt, err := reader.Next() // blocks until the next dispatch or EOF
    if err != nil {
        break
    }
    handle(evt)
}
```

Assertions (`RequireData`, `RequireEventID`, `RequireRetry`, `RequireDataJSON`,
...) and finders (`FindByType`, `FilterByType`) operate on the parsed events.
All `tb`-taking helpers accept [`testing.TB`](https://pkg.go.dev/testing#TB) —
`*testing.T`, `*testing.B`, and Ginkgo's `GinkgoT()`.

## Line-size cap

A single SSE line must fit in [DefaultMaxLineBytes] (1 MiB, terminator
included); over the cap the scan fails with an error wrapping
`bufio.ErrTooLong` (`errors.Is`-matchable). Framing parsers that bound event
sizes themselves should mirror their limit:

```go
sseparse.ReadEvents(r, sseparse.WithMaxLineBytes(myMaxEventBytes))
```

## The conformance corpus as a spec oracle

The Web Platform Tests `eventsource/format-*` corpus, the spec's own example
streams, and Chromium's `event_source_parser_test.cc` cases are exported as
**data** — assert YOUR parser against the same vectors that pin this one, with
no sseparse parsing code in the loop:

```go
func TestParserConformance(t *testing.T) {
    t.Parallel()

    for _, vector := range sseparse.MustCorpus(t) {
        t.Run(vector.Name, func(t *testing.T) {
            t.Parallel()

            got := myParser(vector.Wire) // your framing parser
            requireEqual(t, vector.Events, got)
        })
    }
}
```

Each vector carries the exact wire bytes (`Wire`) and the expected dispatched
events (`Events`: type, joined data payload, last event ID, retry). A vector
with no events pins an input where nothing may dispatch — comments,
id/retry-only frames, a final frame denied its blank line. `Wire` and `Events`
are the framing-only projection a data-payload parser needs.

The corpus JSON (`testdata/wpt_format_corpus.json`) is the single source of
truth: sseparse's own conformance tests load it through `Corpus()`, so parser
and data cannot drift apart.

### Ingesting new WPT vectors

The corpus grows through `gen_corpus.go` (wired via `go:generate`), never by
hand-editing the JSON — the byte-form gate rejects anything the canonical
recipe did not write:

1. Transcribe the upstream test into a vector (name, url, wire, events) in
   `testdata/corpus_pending.json`, same shape as the corpus.
2. Run `go generate .` from `sseparse/`. Every pending vector is executed
   through the real reader first; the declared `Events` must match what
   actually dispatches, or the run aborts and nothing is written.
3. Delete `testdata/corpus_pending.json` and commit the rewritten corpus.

Running `go generate .` with no pending file is a no-op — and the idempotence
proof: a zero diff means the checked-in bytes are still canonical.

## Relationship to ssetest

[ssetest](../ssetest/) layers the HTTP collection helpers (`Collect`,
`CollectN`, `CollectWithTimeout`, request options) on top of this parser and
re-exports its entire API, so handler tests need one import. Import sseparse
directly when you are not testing an HTTP handler.

See [pkg.go.dev](https://pkg.go.dev/github.com/larsartmann/go-sse/sseparse)
for the complete API.
