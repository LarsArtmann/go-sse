package sseparse_test

// This file runs the exported conformance corpus through the reader and
// asserts the exact observable output, so the official suite runs in our CI.
//
// The corpus data (testdata/wpt_format_corpus.json) is the single source of
// truth, loaded via [sseparse.Corpus]: the transcribed Web Platform Tests
// eventsource/format-* vectors (each a precise wire-bytes → expected-events
// pair, with the upstream message.py terminator expansion already applied),
// the spec § 9.2.6 example streams, and Chromium's event_source_parser_test.cc
// unit cases. Third-party parsers consume the same data as a spec oracle
// without running sseparse code in the loop.
//
// Upstream corpus: https://github.com/web-platform-tests/wpt/tree/master/eventsource
// Spec: https://html.spec.whatwg.org/multipage/server-sent-events.html

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-sse/sseparse"
)

// TestWPTFormatCorpus runs every conformance vector through the reader and
// asserts the exact observable output.
func TestWPTFormatCorpus(t *testing.T) {
	t.Parallel()

	for _, tc := range sseparse.MustCorpus(t) {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			events, err := sseparse.ReadEvents(strings.NewReader(tc.Wire))
			if err != nil {
				t.Fatalf("%s: read events: %v", tc.URL, err)
			}

			requireEventsMatch(t, tc, events)
		})
	}
}

// minCorpusVectors is the integrity floor for the exported corpus: well above
// today's 29 vectors' breakdown by family, low enough that pruning a single
// vector for a documented reason does not trip it.
const minCorpusVectors = 25

// TestCorpusIntegrity pins the corpus data itself. An accidentally emptied or
// truncated testdata file would otherwise pass conformance vacuously (no
// vectors, no failures), so every family must stay present and every vector
// named, unique, and sourced.
func TestCorpusIntegrity(t *testing.T) {
	t.Parallel()

	vectors := sseparse.MustCorpus(t)

	if len(vectors) < minCorpusVectors {
		t.Fatalf("corpus shrank to %d vectors (floor %d) — did testdata/ lose a family?", len(vectors), minCorpusVectors)
	}

	provenance := map[string]bool{"wpt:": false, "spec:": false, "chromium:": false}

	seen := make(map[string]bool, len(vectors))

	for _, vector := range vectors {
		if vector.Name == "" {
			t.Error("corpus vector with empty name")
		}

		if seen[vector.Name] {
			t.Errorf("duplicate corpus vector name %q", vector.Name)
		}

		seen[vector.Name] = true

		if vector.Wire == "" {
			t.Errorf("corpus vector %q has empty wire input", vector.Name)
		}

		for prefix := range provenance {
			if strings.HasPrefix(vector.URL, prefix) {
				provenance[prefix] = true
			}
		}
	}

	for prefix, found := range provenance {
		if !found {
			t.Errorf("corpus lost its %s* family (no vector cites such a URL)", prefix)
		}
	}
}

// requireEventsMatch asserts the parsed events equal the vector's expected
// observable output, with a failure message that includes the wire bytes.
func requireEventsMatch(t *testing.T, tc sseparse.ConformanceVector, got []sseparse.Event) {
	t.Helper()

	if len(got) != len(tc.Events) {
		t.Fatalf("%s: event count: got %d, want %d\nwire: %q\ngot:\n%s",
			tc.URL, len(got), len(tc.Events), tc.Wire, sseparse.EventsString(got))
	}

	for i, want := range tc.Events {
		evt := got[i]

		if evt.Type != want.Type {
			t.Errorf("%s: event[%d] type: got %q, want %q", tc.URL, i, evt.Type, want.Type)
		}

		if data := evt.Data(); data != want.Data {
			t.Errorf("%s: event[%d] data: got %q, want %q", tc.URL, i, data, want.Data)
		}

		if evt.ID != want.ID {
			t.Errorf("%s: event[%d] last event id: got %q, want %q", tc.URL, i, evt.ID, want.ID)
		}

		if evt.Retry != want.Retry {
			t.Errorf("%s: event[%d] retry: got %d, want %d", tc.URL, i, evt.Retry, want.Retry)
		}
	}
}
