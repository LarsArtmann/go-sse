package ssetest_test

// ONE-TIME GENERATOR — writes ../sseparse/testdata/wpt_format_corpus.json from
// the Go corpus literals below (wptCorpus/specExamples/chromiumParserCases in
// wpt_format_corpus_test.go). Delete this file after running:
//
//	GOWORK=off go test . -run TestWriteCorpusJSON -count=1
//
// The JSON becomes the single source of truth; sseparse's corpus test loads it
// via go:embed, so parser-vs-corpus conformance keeps running without these
// literals.

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

// corpusNotes carries the explanatory comments from the Go corpus literals
// into the JSON artifact, so the transcribed WPT context survives the move.
var corpusNotes = map[string]string{
	"format-field-data": `A data line with an empty value still dispatches; two bare "data" lines dispatch an event whose payload is a single newline.`,
	"format-field-event-empty":                   `An empty event type fires onmessage, i.e. type "".`,
	"format-field-id-null":                       `An id field whose value contains U+0000 NULL anywhere is ignored entirely (spec § 9.2.6); the last event ID stays "".`,
	"format-field-parsing":                       `Field names are case-sensitive; unknown fields and no-colon lines with unknown names are ignored; only the first colon separates; a single leading space after the colon is stripped (never a tab).`,
	"format-field-retry":                         `Leading zeros are allowed: the value is a decimal integer.`,
	"format-field-retry-bogus":                   `A bogus retry does not reset a previously valid one, and a retry line takes effect even in a frame that never dispatches.`,
	"format-field-retry-empty":                   `Derived from the upstream file, which sends a bare "retry" line and asserts the reconnection time is unchanged: an empty value is not all-digits, so it is ignored. Pinned here with a prior valid value.`,
	"format-field-unknown":                       `Unknown field names (with or without a colon) are ignored.`,
	"format-newlines":                            `CR, LF, and CRLF are all line terminators; a bare "data" line with no colon is an empty data line; "\r\r" is two terminators, the second of which is the blank line that dispatches the frame.`,
	"format-leading-space":                       `Only one leading space is stripped: a tab survives, "data: " is an empty data line.`,
	"format-comments":                            `Comment lines are ignored; note the ":<CR><LF>" comment: the CRLF is one terminator, not a comment line followed by a blank dispatch line.`,
	"format-null-character":                      `NUL is valid data; only in id values is it rejected.`,
	"format-bom":                                 `Exactly one leading UTF-8 BOM is stripped. A mid-stream BOM is NOT: it poisons the field name, so that line is ignored.`,
	"format-bom-2":                               `Two leading BOMs: only the first is stripped; the second poisons the first field name, so only the later events surface.`,
	"format-utf-8":                               `The stream is always decoded as UTF-8 regardless of any charset parameter; non-ASCII payloads pass through unchanged.`,
	"format-data-before-final-empty-line":        `newline=none: the final frame never gets its blank line, so it is discarded at EOF — and its id:test must not leak into the last event ID of any dispatched event.`,
	"spec-example-two-identical-events":          `The spec pairs "data: test" with "data:test" to show that the single space after the colon is optional: both fire the same event.`,
	"LastEventIdShouldNotBeReset":                `The last event ID persists across frames until the next id field: the third event reports "2", and its frame never restated the id.`,
	"LastEventIdCanBeUpdatedEvenWhenDataIsEmpty": `An id line updates the buffer in a frame that never dispatches, and an empty id value resets the buffer to "".`,
	"RetryTakesEffectEvenWhenNotDispatching":     `A retry line in a dataless frame still updates the reconnection time; the next dispatched event reports it.`,
	"NonDigitRetryShouldBeIgnored":               `Values (after the spec's single-space strip) that are not pure ASCII digits are ignored and do not reset a previously set value. " 1234" survives the strip as "1234" and therefore counts — a single space after the colon is stripped per spec § 9.2.6. The final "retry: 1234" line: single space stripped, valid, retry 1234.`,
	"TrailingCREndsLineAtEOF":                    `A lone CR terminates the line even when EOF follows immediately, so this frame's data line is complete — but the blank line required to dispatch it never arrives, and the frame is discarded at EOF.`,
}

type genEvent struct {
	Type  string `json:"type,omitempty"`
	Data  string `json:"data"`
	ID    string `json:"id,omitempty"`
	Retry uint   `json:"retry,omitempty"`
}

type genVector struct {
	Name   string    `json:"name"`
	URL    string    `json:"url,omitempty"`
	Notes  string    `json:"notes,omitempty"`
	Wire   string    `json:"wire"`
	Events []genEvent `json:"events,omitempty"`
}

func TestWriteCorpusJSON(t *testing.T) {
	vectors := make([]genVector, 0, len(allConformanceCases()))

	for _, tc := range allConformanceCases() {
		events := make([]genEvent, 0, len(tc.want))
		for _, want := range tc.want {
			events = append(events, genEvent{
				Type: want.Type, Data: want.Data, ID: want.ID, Retry: want.Retry,
			})
		}

		vectors = append(vectors, genVector{
			Name:   tc.name,
			URL:    tc.url,
			Notes:  corpusNotes[tc.name],
			Wire:   tc.wire,
			Events: events,
		})
	}

	raw, err := json.Marshal(vectors)
	if err != nil {
		t.Fatalf("marshal corpus: %v", err)
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		t.Fatalf("indent corpus: %v", err)
	}

	out := pretty.String() + "\n"
	const path = "../sseparse/testdata/wpt_format_corpus.json"

	if err := os.WriteFile(path, []byte(out), 0o644); err != nil { //nolint:gosec // test fixture, no secrets
		t.Fatalf("write %s: %v", path, err)
	}

	// Round-trip proof: decoding the written file and re-marshaling it must
	// reproduce the canonical encoding byte-for-byte.
	var reload []genVector
	if err := json.Unmarshal([]byte(out), &reload); err != nil {
		t.Fatalf("reload %s: %v", path, err)
	}

	remarshal, err := json.Marshal(reload)
	if err != nil {
		t.Fatalf("remarshal corpus: %v", err)
	}

	if !bytes.Equal(remarshal, raw) {
		t.Fatal("round-trip mismatch: decoded JSON differs from the source vectors")
	}

	t.Logf("wrote %d vectors to %s", len(vectors), path)
}
