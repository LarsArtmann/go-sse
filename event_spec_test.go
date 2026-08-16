package sse_test

// Spec-conformance goldens for the writer side, transcribed from the WHATWG
// HTML Living Standard § 9.2 (Server-Sent Events):
// https://html.spec.whatwg.org/multipage/server-sent-events.html
//
// The reader-side conformance corpus (WPT transcription) lives in the ssetest
// module; this file pins that everything this library WRITES is exactly the
// wire format the spec's examples print.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse"
)

// TestWriteEvent_SpecStockTickerExample pins the spec § 9.2.6 stock-ticker
// example: three data lines dispatch as one event whose payload is the lines
// joined by newlines.
func TestWriteEvent_SpecStockTickerExample(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	if err := sse.WriteEvent(&buf, sse.Event{Data: "YHOO\n+2\n10"}); err != nil {
		t.Fatalf("write event: %v", err)
	}

	const want = "data: YHOO\ndata: +2\ndata: 10\n\n"

	if got := buf.String(); got != want {
		t.Errorf("wire: got %q, want %q", got, want)
	}
}

// TestWriteEvent_SpecSpaceAfterColonIsOptional pins the spec § 9.2.6 note that
// "data: test" and "data:test" produce the same event: the writer emits the
// single-space form, which every conformant reader strips.
func TestWriteEvent_SpecSpaceAfterColonIsOptional(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	if err := sse.WriteEvent(&buf, sse.Event{Data: "test"}); err != nil {
		t.Fatalf("write event: %v", err)
	}

	const want = "data: test\n\n"

	if got := buf.String(); got != want {
		t.Errorf("wire: got %q, want %q", got, want)
	}
}

// TestWriteEvent_SpecFieldOrder pins the writer's field order — event, data,
// id, retry, blank line — which the spec's § 9.2.6 examples consistently use
// and consumers' golden logs expect.
func TestWriteEvent_SpecFieldOrder(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteEvent(&buf, sse.Event{
		Event: "custom event",
		Data:  "third event",
		ID:    sse.NewEventID("3"),
		Retry: 10000,
	})
	if err != nil {
		t.Fatalf("write event: %v", err)
	}

	const want = "event: custom event\n" +
		"data: third event\n" +
		"id: 3\n" +
		"retry: 10000\n" +
		"\n"

	if got := buf.String(); got != want {
		t.Errorf("wire: got %q, want %q", got, want)
	}
}

// TestWriteEvent_SpecEmptyDataValueDispatches pins that empty data writes one
// empty data line — the wire form the spec's format-field-data behavior
// dispatches as an event with an empty payload (a frame with NO data line
// would not dispatch at all).
func TestWriteEvent_SpecEmptyDataValueDispatches(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	if err := sse.WriteEvent(&buf, sse.Event{Data: ""}); err != nil {
		t.Fatalf("write event: %v", err)
	}

	const want = "data: \n\n"

	if got := buf.String(); got != want {
		t.Errorf("wire: got %q, want %q", got, want)
	}
}

// TestWriteEvent_SpecCRAndCRLFSplitIntoDataLines pins § 9.2.5: CR, LF, and
// CRLF are all line terminators, so every line of a payload gets its own
// "data: " prefix — CRLF must not produce a doubled or empty line.
func TestWriteEvent_SpecCRAndCRLFSplitIntoDataLines(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		data string
		want string
	}{
		{name: "lf", data: "a\nb", want: "data: a\ndata: b\n\n"},
		{name: "crlf", data: "a\r\nb", want: "data: a\ndata: b\n\n"},
		{name: "lone cr", data: "a\rb", want: "data: a\ndata: b\n\n"},
		{
			name: "crlf cr lf mix",
			data: "a\r\nb\rc\nd",
			want: "data: a\ndata: b\ndata: c\ndata: d\n\n",
		},
	}

	for _, terminatorCase := range cases {
		t.Run(terminatorCase.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			if err := sse.WriteEvent(&buf, sse.Event{Data: terminatorCase.data}); err != nil {
				t.Fatalf("write event: %v", err)
			}

			if got := buf.String(); got != terminatorCase.want {
				t.Errorf("wire: got %q, want %q", got, terminatorCase.want)
			}
		})
	}
}

// TestWriteHeartbeat_SpecCommentForm pins the keep-alive frame as a comment
// line (§ 9.2.6: "Lines that start with ... U+003A COLON (:) are ignored"),
// followed by the blank line that terminates the comment frame.
func TestWriteHeartbeat_SpecCommentForm(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	if err := sse.WriteHeartbeat(&buf); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	const want = ": heartbeat\n\n"

	if got := buf.String(); got != want {
		t.Errorf("wire: got %q, want %q", got, want)
	}
}

// TestParseEventID_SpecValueSpace pins § 9.2.4: the Last-Event-ID value space
// excludes exactly U+0000 NULL, U+000A LF, and U+000D CR. Everything else —
// unicode, spaces, colons, tabs — is a legal server-defined identifier.
func TestParseEventID_SpecValueSpace(t *testing.T) {
	t.Parallel()

	for _, forbidden := range []string{"\x00", "a\x00b", "\x00x", "x\x00", "\n", "\r", "a\nb", "a\rb"} {
		if _, err := sse.ParseEventID(forbidden); err == nil {
			t.Errorf("ParseEventID(%q): expected rejection, got nil", forbidden)
		}
	}

	for _, legal := range []string{"", "1", "döm", "id with spaces", "colons:are:fine", "\ttab"} {
		id, err := sse.ParseEventID(legal)
		if err != nil {
			t.Errorf("ParseEventID(%q): unexpected error: %v", legal, err)

			continue
		}

		if id.Get() != legal {
			t.Errorf("ParseEventID(%q): got %q", legal, id.Get())
		}
	}
}

// TestParseEventID_RejectionMessageIsActionable ensures the rejection tells
// the caller which forbidden character class was found — this error surfaces
// from handlers parsing the Last-Event-ID request header.
func TestParseEventID_RejectionMessageIsActionable(t *testing.T) {
	t.Parallel()

	_, err := sse.ParseEventID("bad\x00id")
	if err == nil {
		t.Fatal("expected error for NUL in event id")
	}

	if !strings.Contains(err.Error(), "forbidden character") {
		t.Errorf("error should name the problem class, got: %v", err)
	}
}
