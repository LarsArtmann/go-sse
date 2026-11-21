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

// Name is what go-branded-id diagnostics render for [EventID]. It returns a
// string literal (not a constant) because go-branded-id's linter only
// recognizes literal returns and would otherwise re-flag the brand type.
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

// frameOverheadBytes bounds the fixed wire framing beyond the variable-length
// fields: "event: " (8), "data: " (7 per line — extra lines grow the buffer,
// which append handles), "id: " (5), "retry: " + uint digits (up to 27), and
// newlines. Sizing WriteEvent's buffer with it keeps a typical event to one
// allocation.
const frameOverheadBytes = 32

// errEventIDInvalid is returned by [ParseEventID] for malformed values.
// Declared as the error interface (not the concrete *errorfamily.Error) so
// errors.Is call sites match the sentinel guard exactly.
var errEventIDInvalid error = errorfamily.NewRejection(
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
//
// The whole frame is written with a single Write call. A writer that
// accepts only part of the frame (n < len, nil error) gets the remaining
// bytes silently dropped on the wire — an undetectably corrupt SSE stream —
// so it is reported as an error wrapping [io.ErrShortWrite] instead. The
// frame is never retried or re-emitted: bytes already accepted are on the
// wire, and re-sending them would corrupt the frame further.
func WriteEvent(w io.Writer, evt Event) error {
	// Pre-size once instead of growing through append: the frame prefix
	// overhead ("event: ", "data: ", "id: ", "retry: <uint>", newlines) is
	// bounded by 32 bytes past the variable-length fields, so typical events
	// allocate exactly once. A slight underestimate is still correct — append
	// grows — it just costs the old growth reallocations.
	buf := make([]byte, 0, len(evt.Event)+len(evt.Data)+len(evt.ID.Get())+frameOverheadBytes)

	if evt.Event != "" {
		buf = append(buf, 'e', 'v', 'e', 'n', 't', ':', ' ')
		buf = append(buf, evt.Event...)
		buf = append(buf, '\n')
	}

	dataLines := 0

	forEachLine(evt.Data, func(line string) {
		buf = append(buf, 'd', 'a', 't', 'a', ':', ' ')
		buf = append(buf, line...)
		buf = append(buf, '\n')
		dataLines++
	})

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

	n, err := w.Write(buf)
	if err != nil {
		return errorfamily.Wrapf(
			err,
			errorfamily.Transient,
			"sse.write_failed",
			"write sse event %q (%d data bytes, %d lines)",
			evt.Event,
			len(evt.Data),
			dataLines,
		)
	}

	if n != len(buf) {
		return errorfamily.Wrapf(
			io.ErrShortWrite,
			errorfamily.Transient,
			"sse.write_short",
			"short write of sse event %q: %d of %d frame bytes accepted",
			evt.Event,
			n,
			len(buf),
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
