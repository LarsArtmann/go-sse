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
// Use [ParseEventID] to construct from a string (rejects control characters
// and newlines, which would corrupt the SSE wire format).
type EventID = brandid.ID[eventBrand, string]

// NewEventID constructs an [EventID] from a string. Performs no validation —
// use [ParseEventID] for untrusted input (e.g., from request headers).
func NewEventID(s string) EventID { return brandid.NewID[eventBrand](s) }

// base10 is the numeric base for decimal integer formatting.
const base10 = 10

// errEventIDInvalid is returned by [ParseEventID] for malformed values.
var errEventIDInvalid = errorfamily.NewRejection(
	"sse.event_id_invalid",
	"sse event id: contains forbidden character (newline or carriage return)",
)

// ParseEventID converts a string to an [EventID], rejecting values that
// would corrupt the SSE wire format (newlines, carriage returns). Empty strings
// are allowed (representing "no ID" / initial connection).
func ParseEventID(s string) (EventID, error) {
	if strings.ContainsAny(s, "\n\r") {
		return EventID{}, errorfamily.Wrapf(errEventIDInvalid, errorfamily.Rejection,
			"sse.event_id_invalid", "%q", s)
	}

	return NewEventID(s), nil
}

// MustParseEventID is the panicking variant of [ParseEventID] for tests
// and constants. Panics if the input contains newlines.
func MustParseEventID(s string) EventID {
	id, err := ParseEventID(s)
	if err != nil {
		panic(fmt.Errorf("MustParseEventID: %w", err))
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
	// after a connection drop.
	Retry int
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

	for _, line := range splitLines(evt.Data) {
		buf = append(buf, 'd', 'a', 't', 'a', ':', ' ')
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}

	if !evt.ID.IsZero() {
		buf = append(buf, 'i', 'd', ':', ' ')
		buf = append(buf, evt.ID.Get()...)
		buf = append(buf, '\n')
	}

	if evt.Retry > 0 {
		buf = append(buf, 'r', 'e', 't', 'r', 'y', ':', ' ')
		buf = strconv.AppendInt(buf, int64(evt.Retry), base10)
		buf = append(buf, '\n')
	}

	buf = append(buf, '\n')

	if _, err := w.Write(buf); err != nil {
		return errorfamily.Wrapf(
			err,
			errorfamily.Transient,
			"sse.write_failed",
			"write sse event",
		)
	}

	return nil
}

// WriteHeartbeat writes a comment frame (SSE comment line).
// Browsers ignore it, but it keeps the connection alive through
// ALB/Nginx/Cloudflare idle timeouts.
func WriteHeartbeat(w io.Writer) error {
	_, err := w.Write([]byte(": heartbeat\n\n"))

	return err //nolint:wrapcheck // raw write error is already actionable
}

// WriteRetry writes the SSE retry field, telling the browser how many
// milliseconds to wait before reconnecting after a connection drop.
// Per the SSE spec, this is sent once and persists until overwritten.
func WriteRetry(w io.Writer, ms int) error {
	_, err := fmt.Fprintf(w, "retry: %d\n\n", ms)

	return err //nolint:wrapcheck // raw write error is already actionable
}

// splitLines splits a string into lines for SSE data field formatting.
// Each line in the SSE spec must be prefixed with "data: ".
// Fast path: if the data contains no newline, returns a single-element
// slice without allocating a backing array.
func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}

	if !strings.Contains(s, "\n") {
		return []string{s}
	}

	var lines []string

	start := 0

	for i := range len(s) {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}

			lines = append(lines, line)
			start = i + 1
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}
