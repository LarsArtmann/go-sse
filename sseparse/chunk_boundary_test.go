package sseparse_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse/sseparse"
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
		for _, tc := range sseparse.MustCorpus(t) {
			t.Run(tc.Name, func(t *testing.T) {
				t.Parallel()

				whole, err := sseparse.ReadEvents(strings.NewReader(tc.Wire))
				if err != nil {
					t.Fatalf("%s: baseline parse: %v", tc.URL, err)
				}

				chunked, err := sseparse.ReadEvents(
					&chunkedReader{data: []byte(tc.Wire), size: chunkSize},
				)
				if err != nil {
					t.Fatalf("%s: chunked parse (size %d): %v", tc.URL, chunkSize, err)
				}

				if len(chunked) != len(whole) {
					t.Fatalf("%s: chunk size %d: event count: got %d, want %d\nwire: %q",
						tc.URL, chunkSize, len(chunked), len(whole), tc.Wire)
				}

				for i := range whole {
					a, b := whole[i], chunked[i]

					if a.Type != b.Type || a.ID != b.ID || a.Retry != b.Retry ||
						a.Data() != b.Data() {
						t.Fatalf(
							"%s: chunk size %d: event[%d] differs:\nwhole:  %+v\nchunked:%+v\nwire: %q",
							tc.URL,
							chunkSize,
							i,
							a,
							b,
							tc.Wire,
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

		events, err := sseparse.ReadEvents(&chunkedReader{data: []byte(wire), size: 1})
		if err != nil {
			t.Fatalf("read events: %v", err)
		}

		sseparse.RequireEventCount(t, events, 1)

		if got := events[0].Data(); got != "x\ny" {
			t.Errorf("data: got %q, want %q (CRLF must not dispatch early)", got, "x\ny")
		}
	})

	t.Run("lone CR terminates without consuming the next line", func(t *testing.T) {
		t.Parallel()

		// "x\r" terminates the first data line; "data: y" remains its own
		// line — the CR must not swallow the following bytes.
		wire := "data: x\rdata: y\n\n"

		events, err := sseparse.ReadEvents(&chunkedReader{data: []byte(wire), size: 1})
		if err != nil {
			t.Fatalf("read events: %v", err)
		}

		sseparse.RequireEventCount(t, events, 1)

		if got := events[0].Data(); got != "x\ny" {
			t.Errorf("data: got %q, want %q", got, "x\ny")
		}
	})

	t.Run("double CR is a line plus a blank dispatch line", func(t *testing.T) {
		t.Parallel()

		events, err := sseparse.ReadEvents(&chunkedReader{data: []byte("data: x\r\r"), size: 1})
		if err != nil {
			t.Fatalf("read events: %v", err)
		}

		sseparse.RequireEventCount(t, events, 1)
		sseparse.RequireData(t, events[0], "x")
	})
}

// TestParserBOMSplitAcrossReads is the explicit BOM boundary matrix: the
// 3-byte UTF-8 BOM (and the bytes immediately after it) land on every chunk
// boundary from size 1 through 7. The BOM probe consumes the first three bytes
// through its own ReadFull, so short reads inside the BOM are exactly the
// states the 1–4096 sweep covers only implicitly.
func TestParserBOMSplitAcrossReads(t *testing.T) {
	t.Parallel()

	const payload = "data: bom-stripped\n\n"

	tests := []struct {
		name      string
		wire      string
		wantCount int
		wantData  string
	}{
		// One leading BOM is stripped — exactly once — wherever its bytes split.
		{"single bom", "\xEF\xBB\xBF" + payload, 1, "bom-stripped"},
		// A second BOM is NOT stripped: it poisons the field name, the line is
		// ignored, and nothing dispatches.
		{"double bom", "\xEF\xBB\xBF\xEF\xBB\xBF" + payload, 0, ""},
		// A mid-stream BOM poisons the NEXT field name ("\xEF\xBB\xBFdata" is
		// unknown), so the second frame dispatches nothing.
		{"mid-stream bom", payload + "\xEF\xBB\xBFdata: y\n\n", 1, "bom-stripped"},
	}

	for _, tc := range tests {
		for _, chunkSize := range []int{1, 2, 3, 4, 5, 6, 7} {
			t.Run(fmt.Sprintf("%s/chunk-%d", tc.Name, chunkSize), func(t *testing.T) {
				t.Parallel()

				events, err := sseparse.ReadEvents(
					&chunkedReader{data: []byte(tc.Wire), size: chunkSize},
				)
				if err != nil {
					t.Fatalf("%s (chunk %d): read events: %v", tc.Name, chunkSize, err)
				}

				sseparse.RequireEventCount(t, events, tc.wantCount)

				if tc.wantCount > 0 {
					sseparse.RequireData(t, events[0], tc.wantData)
				}
			})
		}
	}
}
