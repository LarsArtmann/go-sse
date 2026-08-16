package ssetest_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-sse/ssetest"
)

// FuzzReadEvents asserts that the SSE wire parser never panics on arbitrary
// input, that its invariants hold for every parseable input, and that the
// result is independent of how the stream is chunked.
func FuzzReadEvents(f *testing.F) {
	seeds := []string{
		"",
		"\n",
		"data: hello\n\n",
		"event: feed\ndata: line\ndata: line\ndata: line\n\n",
		"event: feed\r\nid: 42\r\nretry: 3000\r\ndata: crlf\r\n\r\n",
		": comment only\ndata: x\n\n",
		": heartbeat\n\n",
		"id: 1\n\nid: 2\n\n",
		"retry: 3000\n\n",
		"event: named\nid: 9\nretry: 100\n\n",
		"retry: not-a-number\ndata: x\n\n",
		"data: no trailing newline",
		"fieldwithoutcolon\n\n",
		"data:nospace\n\n",
		// WPT conformance corpus (wpt_format_corpus_test.go has the citations).
		"data:test\r\ndata\ndata:test\r\r\n", // format-newlines
		"data:\ttest\rdata: \ndata:test\n\n", // format-leading-space
		"data:\x00\ndata:  2\rData:1\ndata\x00:2\ndata:1\r\x00data:4\nda-ta:3\rdata_5\ndata:3\rdata:\r\n data:32\ndata:4\n\n", // format-field-parsing
		"data:1\r:\x00\n:\r\ndata:2\n:x\rdata:3\n:data:fail\r:x\ndata:4\n\n",                                                  // format-comments (short comments)
		"data:\n\ndata\ndata\n\ndata:test\n\n",                                                                                // format-field-data
		"data:\x00\n\n",                                                                                                       // format-null-character
		"\xEF\xBB\xBFdata:1\n\n\xEF\xBB\xBFdata:2\n\ndata:3\n\n",                                                              // format-bom
		"\xEF\xBB\xBF\xEF\xBB\xBFdata:1\n\ndata:2\n\n",                                                                        // format-bom-2
		"id:\x00\ndata:x\n\n",                                                                                                 // format-field-id-null
		"id:x\x00\ndata:x\n\n",
		"retry:03000\ndata:x\n\n",                         // format-field-retry
		"retry:3000\nretry:1000x\n\ndata:x\n\n",           // format-field-retry-bogus
		"retry:2000\n\nretry\n\ndata:x\n\n",               // format-field-retry-empty
		"data:1\nfoo\nfoo: bar\ndata:2\n\n",               // format-field-unknown
		"event: \ndata:data\n\n",                          // format-field-event-empty
		"retry:1000\ndata:test1\n\nid:test\ndata:test2\n", // format-data-before-final-empty-line
		"döm\n\ndata:döm\n\n",                             // format-utf-8
		// Sticky last-event-ID (Chromium LastEventIdShouldNotBeReset).
		"data:1\nid:1\n\nid:2\ndata:2\n\ndata:3\n\n",
		"id:9\n\ndata:x\n\nid:\n\ndata:y\n\n",
		// Trailing CR states.
		"data: x\r",
		"data: x\r\r",
		"\r",
		"\r\n",
		"\xEF\xBB\xBF",
		"\xEF\xBB",
		"\xEF",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, wire string) {
		events, err := ssetest.ReadEvents(strings.NewReader(wire))
		if err != nil {
			t.Fatalf("ReadEvents should never fail on an in-memory reader: %v", err)
		}

		// Dataless-frame invariant: every dispatched event carries at least one
		// data line. Comment/id/retry-only frames must never surface as events.
		for _, evt := range events {
			if len(evt.DataLines) == 0 {
				t.Errorf("event dispatched without data lines: %s", evt)
			}

			_ = evt.String()
			_ = evt.Data()
		}

		// Chunk-boundary invariant: parsing the same bytes one read at a time
		// must produce identical events — TCP chunking can never change the
		// parse result. (The sticky-ID property it exercises alongside the
		// dataless-frame rule above is pinned deterministically by the WPT
		// corpus and the Chromium parser cases.)
		chunked, err := ssetest.ReadEvents(&chunkedReader{data: []byte(wire), size: 1})
		if err != nil {
			t.Fatalf("byte-by-byte read failed on %q: %v", wire, err)
		}

		if len(chunked) != len(events) {
			t.Fatalf(
				"byte-by-byte parse of %q: got %d events, want %d",
				wire,
				len(chunked),
				len(events),
			)
		}

		for i := range events {
			a, b := events[i], chunked[i]
			if a.Type != b.Type || a.ID != b.ID || a.Retry != b.Retry || a.Data() != b.Data() {
				t.Fatalf("byte-by-byte parse of %q: event[%d] %+v != %+v", wire, i, b, a)
			}
		}

		// Exact well-formed input must parse to exactly one event. (Substring
		// matching would be wrong: "0data: hello\n\n" contains the substring
		// but is a different field name, so it correctly dispatches nothing.)
		if wire == "data: hello\n\n" && len(events) != 1 {
			t.Errorf("exact well-formed input parsed to %d events, want 1", len(events))
		}
	})
}
