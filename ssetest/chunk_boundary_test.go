package ssetest_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-sse/ssetest"
)

// TestParserChunkBoundaryIndependence runs the entire conformance corpus
// (WPT vectors, spec § 9.2.6 examples, Chromium parser cases) through readers
// that deliver the stream in small fixed-size chunks, asserting byte-for-byte
// identical results to the single-read parse.
//
// This mirrors Chromium's event_source_parser_test.cc EnqueueOneByOne trick:
// the parser must be independent of TCP chunking. Chunk size 1 forces every
// possible boundary state of the CR/LF/CRLF split function (including a CRLF
// pair split across reads and the BOM split across probe reads); size 3 splits
// the BOM exactly; size 2 splits CRLF and the BOM differently again.
func TestParserChunkBoundaryIndependence(t *testing.T) {
	t.Parallel()

	for _, chunkSize := range []int{1, 2, 3, 5, 7, 4096} {
		for _, tc := range allConformanceCases() {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				whole, err := ssetest.ReadEvents(strings.NewReader(tc.wire))
				if err != nil {
					t.Fatalf("%s: baseline parse: %v", tc.url, err)
				}

				chunked, err := ssetest.ReadEvents(
					&chunkedReader{data: []byte(tc.wire), size: chunkSize},
				)
				if err != nil {
					t.Fatalf("%s: chunked parse (size %d): %v", tc.url, chunkSize, err)
				}

				if len(chunked) != len(whole) {
					t.Fatalf("%s: chunk size %d: event count: got %d, want %d\nwire: %q",
						tc.url, chunkSize, len(chunked), len(whole), tc.wire)
				}

				for i := range whole {
					a, b := whole[i], chunked[i]

					if a.Type != b.Type || a.ID != b.ID || a.Retry != b.Retry ||
						a.Data() != b.Data() {
						t.Fatalf(
							"%s: chunk size %d: event[%d] differs:\nwhole:  %+v\nchunked:%+v\nwire: %q",
							tc.url,
							chunkSize,
							i,
							a,
							b,
							tc.wire,
						)
					}
				}
			})
		}
	}
}

// TestParserCRLFSplitAcrossReads targets the split function's trickiest state
// directly: a CR delivered as the final byte of one read, its LF partner in
// the next. The pair must count as ONE terminator (no phantom blank line), and
// the equivalent lone-CR stream must differ exactly by that blank line.
func TestParserCRLFSplitAcrossReads(t *testing.T) {
	t.Parallel()

	t.Run("crlf pair split across reads is one terminator", func(t *testing.T) {
		t.Parallel()

		// "data: x" then CRLF then "data: y" then CRLF: chunk boundaries fall
		// inside both CRLF pairs.
		wire := "data: x\r\ndata: y\r\n\r\n"

		events, err := ssetest.ReadEvents(&chunkedReader{data: []byte(wire), size: 1})
		if err != nil {
			t.Fatalf("read events: %v", err)
		}

		ssetest.RequireEventCount(t, events, 1)

		if got := events[0].Data(); got != "x\ny" {
			t.Errorf("data: got %q, want %q (CRLF must not dispatch early)", got, "x\ny")
		}
	})

	t.Run("lone CR terminates without consuming the next line", func(t *testing.T) {
		t.Parallel()

		// "x\r" terminates the first data line; "data: y" remains its own
		// line — the CR must not swallow the following bytes.
		wire := "data: x\rdata: y\n\n"

		events, err := ssetest.ReadEvents(&chunkedReader{data: []byte(wire), size: 1})
		if err != nil {
			t.Fatalf("read events: %v", err)
		}

		ssetest.RequireEventCount(t, events, 1)

		if got := events[0].Data(); got != "x\ny" {
			t.Errorf("data: got %q, want %q", got, "x\ny")
		}
	})

	t.Run("double CR is a line plus a blank dispatch line", func(t *testing.T) {
		t.Parallel()

		events, err := ssetest.ReadEvents(&chunkedReader{data: []byte("data: x\r\r"), size: 1})
		if err != nil {
			t.Fatalf("read events: %v", err)
		}

		ssetest.RequireEventCount(t, events, 1)
		ssetest.RequireData(t, events[0], "x")
	})
}
