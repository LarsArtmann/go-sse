# WHATWG § 9.2.6 Conformance Program

The library's parsing and writing behavior is pinned to the WHATWG HTML
Living Standard [§ 9.2.6 (event stream interpretation)](https://html.spec.whatwg.org/multipage/server-sent-events.html#parsing-an-event-stream)
— the algorithm a browser's `EventSource` runs — plus the § 9.2.4
Last-Event-ID value space. This document is the living entry point to that
conformance program; the full history lives in the archived planning and
status docs linked at the bottom.

## The three conformance surfaces

| Surface         | Module     | Pinned by                                                                                       |
| --------------- | ---------- | ----------------------------------------------------------------------------------------------- |
| Parser          | `sseparse` | WPT format corpus as data, chunk-boundary independence, line-cap boundary, fuzzing              |
| Writer          | `sse`      | Spec golden vectors (§ 9.2.6 examples), key-based writers, `FuzzWriteEvent`                     |
| Write→read loop | `ssetest`  | `FuzzWriteReadRoundTrip` (everything `WriteEvent` emits must parse back to the identical event) |

Any change to the reader or writer must keep all three surfaces green — they
are the conformance contract, not ordinary tests.

## The WPT corpus (single source of truth)

`sseparse/testdata/wpt_format_corpus.json` holds 29 vectors: transcriptions
of the official WPT `eventsource/format-*` tests, the spec's § 9.2.6
examples, and Chromium's `event_source_parser_test.cc` cases, each carrying
a `url` citation and, where the reason is not obvious from the wire bytes,
a `notes` field explaining the pinned behavior.

- `sseparse.Corpus()` / `sseparse.MustCorpus()` load it as data — for this
  repo's own tests AND for third-party parsers that want to assert against
  the same vectors with no sseparse parsing code in the loop.
- The JSON is byte-canonical-gated (`MarshalIndent`, two-space indent,
  trailing newline) and integrity-floored (`minCorpusVectors`); hand-editing
  fights the gate — use the ingestion generator
  ([procedure](../sseparse/README.md#ingesting-new-wpt-vectors)).
- Every new vector is executed through the real reader before it is allowed
  in; expectations cannot drift from the parser. The generator's
  idempotence, duplicate-skip, and abort-on-mismatch properties are pinned
  by `sseparse/gen_corpus_roundtrip_test.go`.

## Chunk-boundary independence

Real SSE arrives in arbitrary TCP chunks. `sseparse/chunk_boundary_test.go`
re-runs corpus vectors one byte at a time (Chromium's `EnqueueOneByOne`
trick) and at pathological split points, proving the parser's state machine
is independent of how the stream is chopped — vital because the line
splitter is a custom `bufio.SplitFunc` holding back trailing CR bytes.

## Deviations found and fixed (D1–D6)

The 2026-08-16 conformance audit found six deviations from the spec; all
six are fixed and pinned by tests. They are recorded here because each one
is a behavior browsers rely on that a naive Go implementation gets wrong.

| ID | Deviation (before the fix)                                                            | Spec requirement                                                                                     | Pinned by                                                                              |
| -- | ------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| D1 | Lone CR was not a line terminator (`bufio.ScanLines` splits LF only)                  | § 9.2.5: `end-of-line = cr lf / cr / lf`                                                             | `splitSSELines` + corpus `format-newlines`, chunk-boundary suite                       |
| D2 | An incomplete final frame at EOF was dispatched                                       | § 9.2.6: "Once the end of the file is reached, any pending data must be discarded"                   | `TestReadEvents_IncompleteFinalFrameDiscarded`                                         |
| D3 | A leading UTF-8 BOM was not stripped (it poisoned the first field name)               | § 9.2.6: UTF-8 decode strips **one** leading BOM                                                     | corpus `format-bom`, `format-bom-2` (a mid-stream BOM must still poison — also pinned) |
| D4 | An `id:` field containing NUL was accepted                                            | § 9.2.6: if the field value contains U+0000 NULL, ignore the field (the previous ID stays in effect) | corpus `format-field-id-null`                                                          |
| D5 | Last-event-ID was per-frame, not sticky (an event without `id:` reported an empty ID) | § 9.2.6 dispatch step 1: the last event ID buffer persists until the next `id:` field                | corpus `format-field-id*` + Chromium `LastEventIdShouldNotBeReset` transcription       |
| D6 | `ParseEventID` rejected `\n`/`\r` but not NUL                                         | § 9.2.4: the Last-Event-ID value space excludes U+0000 NULL, U+000A LF, and U+000D CR                | `ParseEventID` in `event.go` (forbidden-character error) + fuzzing                     |

Related sticky-state behavior pinned alongside D5: `retry:` is connection
state, not a per-frame field — an event's `Retry` reports the value in
effect at dispatch, and an invalid `retry:` never resets a prior value
(Chromium `RetryTakesEffectEvenWhenNotDispatching`).

## Line-size cap

A single SSE line (terminator included) must fit `DefaultMaxLineBytes`
(1 MiB); at or over the cap the scan fails with an error wrapping
`bufio.ErrTooLong`. The cap is configurable per call (`WithMaxLineBytes`)
and fuzzed (`FuzzReadEvents`'s second int argument), with the boundary
pinned chunk-invariant.

## Extending the program

1. New upstream behavior → transcribe it into `testdata/corpus_pending.json`
   (name, url, wire, events) and run the generator
   ([sseparse/README.md](../sseparse/README.md)).
2. Bump `minCorpusVectors` in `sseparse/wpt_format_corpus_test.go` to track
   the real corpus size.
3. Any reader/writer change must keep the corpus, chunk-boundary, and
   round-trip suites green, plus `verify.sh`'s bench smoke and the CI fuzz
   budget.

## Sources

- Plan: [2026-08-16 conformance-testing plan](planning/archived/2026-08-16_09-21_SUPERB-spec-based-hardcore-sse-conformance-testing.md) (D1–D6 discovery)
- Corpus-as-data extraction: [2026-09-29 status report](status/2026-09-29_07-03_ci-health-vendorhash-repair-pr-closure.md) and the sseparse module split
- Corpus ingestion procedure: [sseparse/README.md](../sseparse/README.md)
