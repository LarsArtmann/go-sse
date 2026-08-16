package ssetest

import (
	"fmt"
	"strings"
)

// Event is an SSE event decoded from the wire format. It preserves the
// individual data: lines so multi-line payloads can be asserted either joined
// ([Event.Data]) or line-by-line ([Event.DataLines]).
//
// Fields:
//   - Type is the SSE event type (the event: field; empty for unnamed events,
//     which browsers deliver as "message").
//   - DataLines are the individual data: lines, values only (no "data: " prefix).
//   - ID is the optional SSE event ID (the id: field).
//   - Retry is the optional reconnection interval in milliseconds (the retry: field).
type Event struct {
	Type      string
	DataLines []string
	ID        string
	Retry     uint
}

// Data returns the event payload with its data: lines rejoined by "\n",
// reconstructing the original multi-line payload.
func (e Event) Data() string {
	return strings.Join(e.DataLines, "\n")
}

// String returns a human-readable debug representation of the event, showing
// the type, event ID (if any), retry (if non-zero), and data line count.
// Useful for debugging test failures and logging.
func (e Event) String() string {
	if e.ID != "" || e.Retry > 0 {
		return fmt.Sprintf(
			"Event{type=%s id=%s retry=%d datalines=%d}",
			e.Type, e.ID, e.Retry, len(e.DataLines),
		)
	}

	return fmt.Sprintf("Event{type=%s datalines=%d}", e.Type, len(e.DataLines))
}

// EventsString returns a multi-line debug representation of an event slice,
// with one Event per line. Useful for logging test failures involving
// multiple events.
func EventsString(events []Event) string {
	if len(events) == 0 {
		return "(no events)"
	}

	var b strings.Builder

	for i, evt := range events {
		if i > 0 {
			b.WriteString("\n")
		}

		b.WriteString(evt.String())
	}

	return b.String()
}
