package sse

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	brandid "github.com/larsartmann/go-branded-id"
	errorfamily "github.com/larsartmann/go-error-family"
)

// eventBrand is the phantom brand type for [EventID], preventing accidental
// cross-assignment with other string-typed IDs.
type eventBrand struct{}

func (eventBrand) Name() string { return "SSEEvent" }

// EventID is a branded identifier for SSE event identifiers (the id: field
// and the Last-Event-ID request header). It prevents accidental cross-assignment
// with other string-typed IDs.
//
// SSE event IDs are arbitrary server-defined strings — they are NOT ULIDs.
// Use [ParseEventID] to construct from a string (rejects NUL and line
// terminators per the spec's Last-Event-ID value space, § 9.2.4).
type EventID = brandid.ID[eventBrand, string]

// NewEventID constructs an [EventID] from a string. Performs no validation —
// use [ParseEventID] for untrusted input (e.g., from request headers).
func NewEventID(s string) EventID { return brandid.NewID[eventBrand](s) }

// base10 is the numeric base for decimal integer formatting.
const base10 = 10

// errEventIDInvalid is returned by [ParseEventID] for malformed values.
var errEventIDInvalid = errorfamily.NewRejection(
	"sse.event_id_invalid",
	"sse event id: contains forbidden character (NUL, newline, or carriage return)",
)

// ParseEventID converts a string to an [EventID], rejecting values outside
// the Last-Event-ID value space the SSE spec allows (§ 9.2.4): U+0000 NULL,
// U+000A LF, and U+000D CR are forbidden — NUL would be silently dropped by
// browser parsers (§ 9.2.6 ignores id fields containing it) and line
// terminators would corrupt the wire format. Empty strings are allowed
// (representing "no ID" / initial connection).
func ParseEventID(s string) (EventID, error) {
	if strings.ContainsAny(s, "\n\r\x00") {
		return EventID{}, errorfamily.Wrapf(errEventIDInvalid, errorfamily.Rejection,
			"sse.event_id_invalid", "%q", s)
	}

	return NewEventID(s), nil
}

// MustParseEventID is the panicking variant of [ParseEventID] for tests
// and constants. Panics if the input contains NUL, newline, or carriage
// return.
func MustParseEventID(s string) EventID {
	id, err := ParseEventID(s)
	if err != nil {
		panic(errorfamily.Wrapf(err, errorfamily.Rejection,
			"sse.event_id_invalid", "MustParseEventID(%q)", s))
	}

	return id
}

// Event represents a single Server-Sent Event.
//
// Per the [SSE spec]:
//   - Event maps to the event: field. If empty, the default message event fires.
//   - ID maps to the id: field. Browsers send it back via Last-Event-ID on reconnect.
//   - Data maps to the data: field. Multi-line data is split so each line gets
//     its own "data:" prefix (required by the spec).
//   - Retry maps to the retry: field, suggesting a reconnection interval in milliseconds.
//
// [SSE spec]: https://html.spec.whatwg.org/multipage/server-sent-events.html
type Event struct {
	// Event is the SSE event name. Must match the client's event listener.
	// For unnamed events, leave empty (the browser default "message" fires).
	Event string

	// Data is the event payload. Multi-line data is supported;
	// each line is prefixed with "data: " per the SSE specification.
	Data string

	// ID is an optional event identifier. The browser sends this as
	// Last-Event-ID on reconnection, enabling replay of missed events.
	ID EventID

	// Retry is an optional reconnection time in milliseconds.
	// Instructs the browser to wait this long before reconnecting
	// after a connection drop. A zero value means "no retry field emitted".
	Retry uint
}

// String returns a compact, human-readable representation of the event for
// logging and debugging. It is NOT the SSE wire format — use [WriteEvent] for
// that. The format omits empty fields so logs stay readable:
//
//	{event:update id:42 retry:3000 data:<div>new</div>}
func (e Event) String() string {
	var b strings.Builder

	b.WriteByte('{')

	if e.Event != "" {
		b.WriteString("event:")
		b.WriteString(e.Event)
	}

	if !e.ID.IsZero() {
		if b.Len() > 1 {
			b.WriteByte(' ')
		}

		b.WriteString("id:")
		b.WriteString(e.ID.Get())
	}

	if e.Retry > 0 {
		if b.Len() > 1 {
			b.WriteByte(' ')
		}

		b.WriteString("retry:")
		b.WriteString(strconv.FormatUint(uint64(e.Retry), base10))
	}

	if e.Data != "" {
		if b.Len() > 1 {
			b.WriteByte(' ')
		}

		b.WriteString("data:")
		b.WriteString(e.Data)
	}

	b.WriteByte('}')

	return b.String()
}

// WriteEvent writes a single SSE event to the writer in the standard
// Server-Sent Events wire format. Uses direct byte appends instead of
// fmt.Fprintf to minimize allocations on the SSE hot path.
func WriteEvent(w io.Writer, evt Event) error {
	var buf []byte

	if evt.Event != "" {
		buf = append(buf, 'e', 'v', 'e', 'n', 't', ':', ' ')
		buf = append(buf, evt.Event...)
		buf = append(buf, '\n')
	}

	dataLines := splitLines(evt.Data)
	for i := range dataLines {
		buf = append(buf, 'd', 'a', 't', 'a', ':', ' ')
		buf = append(buf, dataLines[i]...)
		buf = append(buf, '\n')
	}

	if !evt.ID.IsZero() {
		buf = append(buf, 'i', 'd', ':', ' ')
		buf = append(buf, evt.ID.Get()...)
		buf = append(buf, '\n')
	}

	if evt.Retry > 0 {
		buf = append(buf, 'r', 'e', 't', 'r', 'y', ':', ' ')
		buf = strconv.AppendUint(buf, uint64(evt.Retry), base10)
		buf = append(buf, '\n')
	}

	buf = append(buf, '\n')

	_, err := w.Write(buf)
	if err != nil {
		return errorfamily.Wrapf(
			err,
			errorfamily.Transient,
			"sse.write_failed",
			"write sse event %q (%d data bytes, %d lines)",
			evt.Event,
			len(evt.Data),
			len(dataLines),
		)
	}

	return nil
}

// heartbeatFrame is the SSE comment frame used for keep-alive pings.
// Browsers ignore comment lines, but they reset idle timers on reverse proxies.
const heartbeatFrame = ": heartbeat\n\n"

// WriteHeartbeat writes a comment frame (SSE comment line).
// Browsers ignore it, but it keeps the connection alive through
// ALB/Nginx/Cloudflare idle timeouts.
func WriteHeartbeat(w io.Writer) error {
	_, err := w.Write([]byte(heartbeatFrame))

	return err //nolint:wrapcheck // raw write error is already actionable
}

// WriteRetry writes the SSE retry field, telling the browser how many
// milliseconds to wait before reconnecting after a connection drop.
// Per the SSE spec, this is sent once and persists until overwritten.
func WriteRetry(w io.Writer, ms uint) error {
	_, err := fmt.Fprintf(w, "retry: %d\n\n", ms)

	return err //nolint:wrapcheck // raw write error is already actionable
}

// JoinLines joins lines with "\n", producing the data string for a multi-line
// [Event]. This is the inverse of [splitLines] at the data level: [WriteEvent]
// will split the result back into individual "data:" lines.
//
// Use [JoinLines] to compose multiple keyed data lines (e.g., DataStar wire
// format) without manual string concatenation:
//
//	evt := sse.Event{
//	    Event: "datastar-patch-elements",
//	    Data:  sse.JoinLines("selector #feed", "mode inner", sse.KeyedLines("elements", html)),
//	}
//
// For the stream-level convenience, see [Stream.SendLines].
func JoinLines(lines ...string) string {
	return strings.Join(lines, "\n")
}

// KeyedLines prefixes every line of value with "key ", producing the
// newline-joined string that [WriteEvent] splits into individual "data:" lines.
//
// This is the building block for SSE protocols that use keyed data lines —
// most notably [DataStar], whose wire format repeats a key prefix on each line
// for multi-line values:
//
//	data: elements <div>
//	data: elements   <span>Hello</span>
//	data: elements </div>
//
// Producing that output:
//
//	html := "<div>\n  <span>Hello</span>\n</div>"
//	sse.KeyedLines("elements", html)
//	// → "elements <div>\nelements   <span>Hello</span>\nelements </div>"
//
// Assign the result to [Event.Data] alongside other keyed lines:
//
//	evt := sse.Event{
//	    Event: "datastar-patch-elements",
//	    Data:  "selector #feed\nmode inner\n" + sse.KeyedLines("elements", html),
//	}
//
// Returns "" when value is empty (no data line emitted).
// An empty key is a no-op — each line gets just a space prefix. This is a
// caller bug (keyed data lines require a key), not a valid pattern.
//
// Line endings in value (LF, CRLF, and lone CR) are normalized to LF by the
// underlying [splitLines] — Windows-style CRLF in HTML fragments is handled
// correctly without producing empty or doubled data lines.
//
// [DataStar]: https://data-star.dev
func KeyedLines(key, value string) string {
	if value == "" {
		return ""
	}

	lines := splitLines(value)

	// Grow hint accounts for each line's key prefix (len(key) + space) and
	// the "\n" separator between lines. Counting the separator for all lines
	// (including the last) is a safe upper bound — at most 1 byte over-allocated.
	const (
		keyValueSepWidth = 1 // " " between key and value
		newlineWidth     = 1 // "\n" between lines
	)

	var b strings.Builder
	b.Grow(len(value) + len(lines)*(len(key)+keyValueSepWidth+newlineWidth))

	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}

		b.WriteString(key)
		b.WriteByte(' ')
		b.WriteString(line)
	}

	return b.String()
}

// WriteKeyedLines writes a single-key SSE event to the writer, prefixing every
// line of value with "key ". This is the wire-only counterpart to
// [Stream.SendKeyed] — for consumers that use [WriteEvent] directly without
// a [Stream] (e.g., custom HTTP scaffolding).
//
// For events with multiple keyed data lines (e.g., DataStar patch-elements with
// selector + mode + elements), compose [KeyedLines] calls into [Event.Data]
// and use [WriteEvent] directly:
//
//	sse.WriteEvent(w, sse.Event{
//	    Event: "datastar-patch-elements",
//	    Data:  "selector #feed\nmode inner\n" + sse.KeyedLines("elements", html),
//	})
func WriteKeyedLines(w io.Writer, eventType, key, value string) error {
	return WriteEvent(w, Event{Event: eventType, Data: KeyedLines(key, value)})
}

// splitLines splits a string into lines for SSE data field formatting.
// Each line in the SSE spec must be prefixed with "data: ".
// Per the SSE spec, CR (\r), LF (\n), and CRLF (\r\n) are all valid line endings.
// Fast path: if the data contains no CR or LF, returns a single-element
// slice without allocating a backing array.
func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}

	if !strings.ContainsAny(s, "\n\r") {
		return []string{s}
	}

	var lines []string

	start := 0

	for i := 0; i < len(s); {
		switch s[i] {
		case '\r':
			lines = append(lines, s[start:i])
			i++

			if i < len(s) && s[i] == '\n' {
				i++ // CRLF: consume both characters as one line break
			}

			start = i
		case '\n':
			lines = append(lines, s[start:i])
			i++
			start = i
		default:
			i++
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}
