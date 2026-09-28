package sseparse

import (
	_ "embed"
	"encoding/json/v2"
	"fmt"
	"testing"
)

// corpusJSON is the exported conformance corpus: the transcribed WPT
// eventsource/format-* vectors, the spec § 9.2.6 example streams, and the
// Chromium event_source_parser_test.cc cases, serialized as data so any SSE
// parser can assert against the same vectors that pin this package.
//
//go:embed testdata/wpt_format_corpus.json
var corpusJSON []byte

// ConformanceVector is one wire-format conformance vector: the exact bytes on
// the wire, and the events a spec-conformant parser must dispatch for them.
//
// A vector with no Events pins an input where nothing must dispatch
// (comment-only streams, id/retry-only frames, a final frame that never got
// its blank line) — for framing-only parsers, the projection "which frames
// dispatch, with which data payload" is exactly Events.
type ConformanceVector struct {
	// Name identifies the vector (the upstream test name for WPT
	// transcriptions, the Chromium test name for parser cases).
	Name string `json:"name"`
	// URL cites the upstream source this vector was transcribed (or derived)
	// from, e.g. "wpt:eventsource/format-bom.any.js".
	URL string `json:"url,omitempty"`
	// Notes explains the conformance behavior the vector pins, where the
	// reason is not obvious from the wire bytes alone.
	Notes string `json:"notes,omitempty"`
	// Wire is the exact byte stream, with terminators already expanded the
	// way the upstream test server sends them.
	Wire string `json:"wire"`
	// Events are the events a conformant parser dispatches for Wire, in
	// order. Data is the payload with data: lines rejoined by "\n".
	Events []ExpectedEvent `json:"events,omitempty"`
}

// ExpectedEvent is the observable surface of one dispatched SSE event: the
// type, the joined data payload, the last event ID in effect, and the
// reconnection time in effect. This mirrors exactly what WPT asserts on
// MessageEvent (type, data, lastEventId) plus the retry field the suite
// observes through reconnection timing.
type ExpectedEvent struct {
	Type  string `json:"type,omitempty"`
	Data  string `json:"data"`
	ID    string `json:"id,omitempty"`
	Retry uint   `json:"retry,omitempty"`
}

// Corpus returns the conformance corpus as data. It is the single source of
// truth for this package's own conformance tests (wpt_format_corpus_test.go
// and chunk_boundary_test.go load it through this function), and it exists so
// third-party SSE parsers can assert against the same vectors — running their
// own parser over each vector's [ConformanceVector.Wire] and comparing the
// dispatched events to [ConformanceVector.Events] — with no sseparse parsing
// code in the loop.
//
// The returned slice is freshly decoded on every call; callers may freely
// range, sort, or copy it.
func Corpus() ([]ConformanceVector, error) {
	var vectors []ConformanceVector

	if err := json.Unmarshal(corpusJSON, &vectors); err != nil {
		return nil, fmt.Errorf("decode embedded WPT format corpus: %w", err)
	}

	return vectors, nil
}

// MustCorpus is like [Corpus] but calls tb.Fatal on error. Accepts
// [testing.TB], so it works with *testing.T, *testing.B, and GinkgoT(). The
// embedded corpus is fixed at compile time, so in practice this never fails;
// it fails only if the checked-in JSON is corrupt, which CI catches.
func MustCorpus(tb testing.TB) []ConformanceVector {
	tb.Helper()

	vectors, err := Corpus()
	if err != nil {
		tb.Fatalf("load WPT format corpus: %v", err)
	}

	return vectors
}
