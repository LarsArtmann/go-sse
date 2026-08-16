package ssetest_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse"
	"github.com/larsartmann/go-sse/ssetest"
)

// TestWriteReadRoundTrip closes the conformance loop: everything the root
// library writes must be exactly what browsers (and therefore ssetest, which
// is pinned to the WPT corpus) read back.
//
// Known, spec-mandated normalizations on the way through the wire:
//   - CR, LF, and CRLF inside Data split into separate data: lines, so the
//     rejoined payload uses LF endings only.
//   - Retry is emitted by the writer only when non-zero, and the reader
//     reports it as connection-level state on the dispatched event.
//   - Events with no data never dispatch (spec § 9.2.6), so a root Event with
//     empty Data surfaces as zero events — covered separately below.
func TestWriteReadRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		evt  sse.Event
		want []wantEvent
	}{
		{
			name: "plain data",
			evt:  sse.Event{Data: "hello"},
			want: []wantEvent{{Data: "hello"}},
		},
		{
			name: "empty data still dispatches one event",
			evt:  sse.Event{Data: ""},
			want: []wantEvent{{Data: ""}},
		},
		{
			name: "named event",
			evt:  sse.Event{Event: "update", Data: "payload"},
			want: []wantEvent{{Type: "update", Data: "payload"}},
		},
		{
			name: "multi-line data rejoins with LF",
			evt:  sse.Event{Data: "YHOO\n+2\n10"},
			want: []wantEvent{{Data: "YHOO\n+2\n10"}},
		},
		{
			name: "CRLF in data normalizes to LF",
			evt:  sse.Event{Data: "windows\r\nline endings"},
			want: []wantEvent{{Data: "windows\nline endings"}},
		},
		{
			name: "lone CR in data normalizes to LF",
			evt:  sse.Event{Data: "old mac\rline"},
			want: []wantEvent{{Data: "old mac\nline"}},
		},
		{
			name: "trailing LF in data is structural, not payload",
			// The spec strips one trailing LF from the data buffer at
			// dispatch, and the writer's splitLines yields no final empty
			// line for a trailing terminator — so "x\n" and "x" are the
			// same wire frame. A trailing newline in Data is not
			// representable in SSE payloads.
			evt:  sse.Event{Data: "x\n"},
			want: []wantEvent{{Data: "x"}},
		},
		{
			name: "event with id",
			evt:  sse.Event{Data: "x", ID: sse.NewEventID("42")},
			want: []wantEvent{{Data: "x", ID: "42"}},
		},
		{
			name: "event with retry",
			evt:  sse.Event{Data: "x", Retry: 3000},
			want: []wantEvent{{Data: "x", Retry: 3000}},
		},
		{
			name: "full event",
			evt: sse.Event{
				Event: "feed",
				Data:  "first line\nsecond line",
				ID:    sse.NewEventID("7"),
				Retry: 5000,
			},
			want: []wantEvent{
				{Type: "feed", Data: "first line\nsecond line", ID: "7", Retry: 5000},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			if err := sse.WriteEvent(&buf, tc.evt); err != nil {
				t.Fatalf("write event: %v", err)
			}

			events := ssetest.MustReadEvents(t, &buf)
			if len(events) != len(tc.want) {
				t.Fatalf(
					"event count: got %d, want %d (wire %q)",
					len(events),
					len(tc.want),
					buf.String(),
				)
			}

			for i, want := range tc.want {
				if events[i].Type != want.Type || events[i].Data() != want.Data ||
					events[i].ID != want.ID || events[i].Retry != want.Retry {
					t.Fatalf(
						"event[%d]: got %+v, want %+v (wire %q)",
						i,
						events[i],
						want,
						buf.String(),
					)
				}
			}
		})
	}
}

// TestWriteReadRoundTrip_HeartbeatAndRetryFramesAreInvisible pins that the
// writer's comment frames (WriteHeartbeat) and dataless retry frames
// (WriteRetry) produce no events on the reader side — while WriteRetry's
// reconnection time still reaches a subsequent event, per the sticky-retry
// rule.
func TestWriteReadRoundTrip_HeartbeatAndRetryFramesAreInvisible(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	if err := sse.WriteHeartbeat(&buf); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	if err := sse.WriteRetry(&buf, 2500); err != nil {
		t.Fatalf("write retry: %v", err)
	}

	if err := sse.WriteEvent(&buf, sse.Event{Data: "visible"}); err != nil {
		t.Fatalf("write event: %v", err)
	}

	events := ssetest.MustReadEvents(t, &buf)
	ssetest.RequireEventCount(t, events, 1)
	ssetest.RequireData(t, events[0], "visible")
	ssetest.RequireRetry(t, events[0], 2500)
}

// TestWriteReadRoundTrip_EmptyDataIsOneEmptyEvent pins the writer/reader
// contract edge: a root Event with empty Data still writes one empty data
// line ("data: \n\n"), which dispatches exactly one empty-payload event —
// the same behavior WPT format-field-data pins for empty data values.
// Genuinely dataless frames (comments via WriteHeartbeat, retry-only frames
// via WriteRetry) never dispatch; they are covered above.
func TestWriteReadRoundTrip_EmptyDataIsOneEmptyEvent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	if err := sse.WriteEvent(&buf, sse.Event{Event: "x", ID: sse.NewEventID("9")}); err != nil {
		t.Fatalf("write event: %v", err)
	}

	events := ssetest.MustReadEvents(t, &buf)
	ssetest.RequireEventCount(t, events, 1)
	ssetest.RequireEventType(t, events[0], "x")
	ssetest.RequireData(t, events[0], "")
	ssetest.RequireEventID(t, events[0], "9")
}

// normalizeLineEndings returns s with every CRLF and lone CR replaced by LF,
// mirroring the writer's splitLines → reader rejoin normalization.
func normalizeLineEndings(s string) string {
	if !strings.ContainsAny(s, "\r") {
		return s
	}

	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

// stripWireUnsafe removes characters that cannot round-trip through the wire
// format in event names and IDs (line terminators, and NUL for IDs per spec
// § 9.2.4/§ 9.2.6).
func stripWireUnsafe(s string, alsoNUL bool) string {
	var b strings.Builder

	for _, r := range s {
		switch r {
		case '\r', '\n':
			continue
		case '\x00':
			if alsoNUL {
				continue
			}
		}

		b.WriteRune(r)
	}

	return b.String()
}

// FuzzWriteReadRoundTrip generates arbitrary root events, writes them, and
// asserts the reader reconstructs the exact observable event — modulo the
// documented CR/CRLF→LF normalization in Data.
func FuzzWriteReadRoundTrip(f *testing.F) {
	type seedEvent struct {
		event string
		data  string
		id    string
		retry uint16
	}

	seeds := []seedEvent{
		{"", "", "", 0},
		{"message", "hello", "1", 0},
		{"update", "a\nb\nc", "42", 3000},
		{"custom", "tab\there", "id with spaces", 1},
		{"", "windows\r\nendings", "", 0},
		{"feed", "", "7", 99},
	}

	for _, s := range seeds {
		f.Add(s.event, s.data, s.id, s.retry)
	}

	f.Fuzz(func(t *testing.T, event, data, id string, retry uint16) {
		evt := sse.Event{
			Event: stripWireUnsafe(event, false),
			Data:  data,
			ID:    sse.NewEventID(stripWireUnsafe(id, true)),
			Retry: uint(retry),
		}

		var buf bytes.Buffer
		if err := sse.WriteEvent(&buf, evt); err != nil {
			t.Fatalf("write event: %v", err)
		}

		wire := buf.String() // snapshot before the reader consumes the buffer
		events := ssetest.MustReadEvents(t, &buf)

		// Empty Data still writes one empty data line, so exactly one
		// empty-payload event comes back (identity, not zero events).
		if data == "" {
			if len(events) != 1 {
				t.Fatalf(
					"empty-data event dispatched %d events, want 1 (wire %q)",
					len(events),
					wire,
				)
			}

			if events[0].Data() != "" {
				t.Errorf(
					"empty-data event payload: got %q, want empty (wire %q)",
					events[0].Data(),
					wire,
				)
			}

			return
		}

		if len(events) != 1 {
			t.Fatalf("got %d events, want 1 (wire %q)", len(events), wire)
		}

		got := events[0]

		if got.Type != evt.Event {
			t.Errorf("type: got %q, want %q (wire %q)", got.Type, evt.Event, wire)
		}

		// CR/CRLF normalize to LF, and one trailing LF is structural (the spec
		// strips it from the data buffer at dispatch; splitLines emits no final
		// empty line), so it does not survive the round trip.
		want := strings.TrimSuffix(normalizeLineEndings(data), "\n")
		if got.Data() != want {
			t.Errorf("data: got %q, want %q (wire %q)", got.Data(), want, wire)
		}

		if got.ID != evt.ID.Get() {
			t.Errorf("id: got %q, want %q (wire %q)", got.ID, evt.ID.Get(), wire)
		}

		if got.Retry != evt.Retry {
			t.Errorf("retry: got %d, want %d (wire %q)", got.Retry, evt.Retry, wire)
		}
	})
}
