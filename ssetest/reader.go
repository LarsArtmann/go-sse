package ssetest

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	maxLineBytes   = 1024 * 1024 // 1 MiB max single line
	initialLineCap = 64 * 1024   // 64 KiB initial buffer
)

// ReadEvents parses the SSE wire format from r and returns all decoded events.
// It reads until EOF, so the source must close or end the stream (e.g., an HTTP
// response body from a handler that sends all events and returns).
//
// The parser handles the standard SSE fields: event, data, id, retry, and
// comment lines (starting with ":"). Each blank line dispatches the current
// event. Per the SSE specification, only frames containing at least one data
// line dispatch — comment frames (e.g., go-sse heartbeats) and id/retry-only
// frames never surface as events. An event without a trailing blank line at
// EOF is still returned, and lines may end with LF or CRLF.
//
// Individual data: lines are preserved in [Event.DataLines] with their values
// only (no "data: " prefix); use [Event.Data] for the rejoined payload.
func ReadEvents(r io.Reader) ([]Event, error) {
	scanner := newSSEScanner(r)

	var (
		events  []Event
		current Event
	)

	for scanner.Scan() {
		// bufio.ScanLines already strips a trailing \r, so CRLF needs no handling here.
		line := scanner.Text()

		if line == "" {
			dispatchFrame(&events, &current)

			continue
		}

		applySSELine(&current, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, errorfamily.WrapTransient(err, CodeSSEScanFailed, "scan SSE stream")
	}

	dispatchFrame(&events, &current)

	return events, nil
}

// dispatchFrame appends current to events if the frame carries data lines,
// then resets the accumulator. Frames without data (comments, id/retry-only
// frames) never dispatch, matching browser behavior.
func dispatchFrame(events *[]Event, current *Event) {
	if len(current.DataLines) > 0 {
		*events = append(*events, *current)
	}

	*current = Event{}
}

// ReadNEvents reads up to count events from r. Returns as soon as count events
// have been dispatched, without waiting for EOF. This is the streaming-reader
// counterpart to [ReadEvents]: use it with a live SSE connection body that
// does not close on its own (e.g., a handler broadcasting through a
// Broadcaster).
//
// A scanner error after events have been collected is treated as a clean
// connection close, not a failure.
func ReadNEvents(r io.Reader, count int) ([]Event, error) {
	if count <= 0 {
		return nil, nil
	}

	scanner := newSSEScanner(r)

	var (
		events  []Event
		current Event
	)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			dispatchFrame(&events, &current)

			if len(events) >= count {
				return events, nil
			}

			continue
		}

		applySSELine(&current, line)
	}

	if err := scanner.Err(); err != nil {
		if len(events) > 0 {
			return events, nil
		}

		return nil, errorfamily.WrapTransient(err, CodeSSEScanFailed, "scan SSE stream")
	}

	dispatchFrame(&events, &current)

	return events, nil
}

// MustReadEvents is like [ReadEvents] but calls t.Fatal on error. Accepts
// [testing.TB], so it works with *testing.T, *testing.B, and GinkgoT().
func MustReadEvents(tb testing.TB, r io.Reader) []Event {
	tb.Helper()

	events, err := ReadEvents(r)
	if err != nil {
		tb.Fatalf("read SSE events: %v", err)
	}

	return events
}

// MustReadNEvents is like [ReadNEvents] but calls t.Fatal on error.
// Use this with streaming SSE connections that do not close on their own.
// Accepts [testing.TB], so it works with *testing.T, *testing.B, and GinkgoT().
func MustReadNEvents(tb testing.TB, r io.Reader, count int) []Event {
	tb.Helper()

	events, err := ReadNEvents(r, count)
	if err != nil {
		tb.Fatalf("read %d SSE events: %v", count, err)
	}

	return events
}

// applySSELine parses a single SSE wire line and folds it into the event.
func applySSELine(evt *Event, line string) {
	if strings.HasPrefix(line, ":") {
		return // SSE comment, ignore
	}

	field, value := parseSSEField(line)

	switch field {
	case "event":
		evt.Type = value
	case "data":
		evt.DataLines = append(evt.DataLines, value)
	case "id":
		evt.ID = value
	case "retry":
		if ms, err := strconv.ParseUint(value, 10, 32); err == nil {
			evt.Retry = uint(ms)
		}
	}
}

// newSSEScanner creates a bufio.Scanner configured for SSE wire-format parsing
// with the package's standard buffer sizes.
func newSSEScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, initialLineCap), maxLineBytes)

	return scanner
}

// parseSSEField splits an SSE line into field name and value. Per the SSE spec,
// the value is everything after the first colon, with a single leading space
// stripped if present. Lines without a colon produce the full line as the field
// with an empty value.
func parseSSEField(line string) (string, string) {
	field, value, found := strings.Cut(line, ":")
	if !found {
		return line, ""
	}

	return field, strings.TrimPrefix(value, " ")
}
