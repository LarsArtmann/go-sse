package ssetest

import (
	"bufio"
	"bytes"
	"errors"
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

// utf8BOM is the UTF-8 byte-order mark (U+FEFF). The SSE spec decodes the
// stream with UTF-8 decode, which strips exactly one leading BOM.
var utf8BOM = [...]byte{0xEF, 0xBB, 0xBF} //nolint:gochecknoglobals // byte arrays cannot be const

// ReadEvents parses the SSE wire format from r and returns all decoded events.
// It reads until EOF, so the source must close or end the stream (e.g., an HTTP
// response body from a handler that sends all events and returns).
//
// The parser implements the WHATWG HTML Living Standard § 9.2.6 event-stream
// interpretation (conformance is pinned by the Web Platform Tests vectors in
// wpt_format_corpus_test.go):
//
//   - Lines end with CR, LF, or CRLF (§ 9.2.5 end-of-line).
//   - Exactly one leading UTF-8 BOM is stripped; a mid-stream BOM is data.
//   - Field names are case-sensitive; unknown fields and ":" comments are ignored.
//   - A single leading space after the ":" separator is stripped (never a tab).
//   - Only frames with at least one data: line dispatch; a "data:" line with an
//     empty value still dispatches an event with an empty payload.
//   - The last event ID is sticky: an id: field updates the buffer, the buffer
//     persists across frames, and each dispatched event reports the buffer's
//     value at dispatch time. An id: value containing U+0000 NULL is ignored.
//   - The retry: value must be all ASCII digits (leading zeros allowed); an
//     invalid value is ignored without resetting a previous one. Like the last
//     event ID, the reconnection time is connection-level state: it persists
//     across frames, and each dispatched event reports the value in effect.
//   - An incomplete final frame (no blank line before EOF) is discarded, per
//     "Once the end of the file is reached, any pending data must be discarded".
//
// Individual data: lines are preserved in [Event.DataLines] with their values
// only (no "data: " prefix); use [Event.Data] for the rejoined payload.
func ReadEvents(r io.Reader) ([]Event, error) {
	parser := streamParser{} //nolint:exhaustruct // zero value is the initial parser state
	scanner := newSSEScanner(r)

	for scanner.Scan() {
		parser.acceptLine(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, errorfamily.WrapTransient(err, CodeSSEScanFailed, "scan SSE stream")
	}

	return parser.events, nil
}

// ReadNEvents reads up to count events from r. Returns as soon as count events
// have been dispatched, without waiting for EOF. This is the streaming-reader
// counterpart to [ReadEvents]: use it with a live SSE connection body that
// does not close on its own (e.g., a handler broadcasting through a
// Broadcaster).
//
// Wire-format semantics are identical to [ReadEvents] (spec § 9.2.6), except
// that reading stops early: a frame still pending when count is reached is
// naturally discarded, as is a frame pending at EOF.
//
// A scanner error after events have been collected is treated as a clean
// connection close, not a failure.
func ReadNEvents(r io.Reader, count int) ([]Event, error) {
	if count <= 0 {
		return nil, nil
	}

	parser := streamParser{} //nolint:exhaustruct // zero value is the initial parser state
	scanner := newSSEScanner(r)

	for scanner.Scan() {
		parser.acceptLine(scanner.Text())

		if len(parser.events) >= count {
			return parser.events, nil
		}
	}

	if err := scanner.Err(); err != nil {
		if len(parser.events) > 0 {
			return parser.events, nil
		}

		return nil, errorfamily.WrapTransient(err, CodeSSEScanFailed, "scan SSE stream")
	}

	return parser.events, nil
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

// streamParser holds the spec-mandated parsing state of one event stream.
//
// It separates per-frame state (the event type and data lines under
// construction, reset at every blank line) from connection-level state: the
// last event ID buffer and the reconnection time, which persist across frames
// until the next id:/retry: field. Each dispatched event snapshots the
// connection-level state, exactly as browsers attach lastEventId and the
// reconnection time to dispatched MessageEvents.
type streamParser struct {
	events []Event
	frame  Event // per-frame accumulator: Type and DataLines only

	lastID string // sticky last event ID buffer (spec § 9.2.6)
	retry  uint   // sticky reconnection time in milliseconds
}

// acceptLine feeds one wire line (terminator stripped) into the parser. An
// empty line dispatches the frame under construction.
func (p *streamParser) acceptLine(line string) {
	if line == "" {
		p.dispatchFrame()

		return
	}

	p.applyField(line)
}

// dispatchFrame appends the frame to the events if it carries data lines, then
// resets the per-frame accumulator. Frames without data (comments, id/retry-only
// frames) never dispatch, matching browser behavior — but their id:/retry:
// fields have already updated the sticky buffers, so they still take effect on
// the next dispatched event.
func (p *streamParser) dispatchFrame() {
	if len(p.frame.DataLines) > 0 {
		p.frame.ID = p.lastID
		p.frame.Retry = p.retry
		p.events = append(p.events, p.frame)
	}

	p.frame = Event{}
}

// applyField parses a single SSE wire line and folds it into the parser state.
func (p *streamParser) applyField(line string) {
	if strings.HasPrefix(line, ":") {
		return // SSE comment, ignore
	}

	field, value := parseSSEField(line)

	switch field {
	case "event":
		p.frame.Type = value
	case "data":
		p.frame.DataLines = append(p.frame.DataLines, value)
	case "id":
		// Spec § 9.2.6: "If the field value does not contain U+0000 NULL, then
		// set the last event ID buffer to the field value. Otherwise, ignore
		// the field." An empty value resets the buffer to "".
		if !strings.ContainsRune(value, '\x00') {
			p.lastID = value
		}
	case "retry":
		// Spec § 9.2.6: only an all-ASCII-digit value (leading zeros allowed)
		// updates the reconnection time; anything else is ignored without
		// resetting a previously set value. 64-bit width so full millisecond
		// ranges parse on every platform.
		if ms, err := strconv.ParseUint(value, 10, 64); err == nil {
			p.retry = uint(ms)
		}
	}
}

// newSSEScanner creates a bufio.Scanner for SSE wire-format parsing: lines are
// split on CR, LF, or CRLF, and a single leading UTF-8 BOM is stripped.
func newSSEScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(stripLeadingBOM(r))
	scanner.Buffer(make([]byte, 0, initialLineCap), maxLineBytes)
	scanner.Split(splitSSELines)

	return scanner
}

// crlfLen is the number of bytes consumed when a CR is followed by its LF
// partner: a CRLF pair counts as one end-of-line, not two.
const crlfLen = 2

// splitSSELines is a bufio.SplitFunc that splits a byte stream into lines on
// CR, LF, or CRLF — the three terminators the SSE spec allows (§ 9.2.5:
// "end-of-line = cr lf / cr / lf"). bufio.ScanLines is not sufficient: it only
// splits on LF, so a lone CR would be swallowed into the line instead of
// terminating it.
//
// A CR as the last buffered byte is held back until one more byte arrives (or
// EOF) so a CRLF pair is always recognized as a single terminator.
func splitSSELines(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	for i := range data {
		switch data[i] {
		case '\n':
			return i + 1, data[:i], nil
		case '\r':
			if i+1 < len(data) {
				if data[i+1] == '\n' {
					return i + crlfLen, data[:i], nil // CRLF: one terminator
				}

				return i + 1, data[:i], nil // lone CR
			}

			if atEOF {
				return i + 1, data[:i], nil // lone CR at EOF
			}

			return 0, nil, nil // trailing CR: wait for one more byte
		}
	}

	if atEOF {
		return len(data), data, nil // final unterminated line
	}

	return 0, nil, nil // request more data
}

// stripLeadingBOM wraps r so that exactly one leading UTF-8 byte-order mark is
// removed before parsing. The spec decodes the stream with UTF-8 decode, which
// strips a single leading U+FEFF; the WPT format-bom and format-bom-2 vectors
// pin that a second (mid-stream) BOM is NOT stripped — it poisons the first
// field name, so the line is ignored as an unknown field.
func stripLeadingBOM(r io.Reader) io.Reader {
	return &bomStripReader{r: r} //nolint:exhaustruct // zero values are correct for the rest
}

// bomStripReader probes the first bytes of the underlying reader once, drops
// them if they form a complete BOM, and otherwise replays them unchanged.
type bomStripReader struct {
	r       io.Reader
	pending []byte // probed bytes that must still be handed out
	checked bool
	err     error // deferred probe error, returned once pending drains
}

func (b *bomStripReader) Read(buf []byte) (int, error) {
	if !b.checked {
		b.probe()
	}

	if len(b.pending) > 0 {
		n := copy(buf, b.pending)
		b.pending = b.pending[n:]

		return n, nil
	}

	if b.err != nil {
		err := b.err
		b.err = nil

		return 0, err
	}

	return b.r.Read(buf) //nolint:wrapcheck // transparent passthrough to the wrapped reader
}

// probe reads up to three bytes and decides whether they are a BOM. Short
// reads are replayed unchanged: only a complete leading BOM is stripped.
// A hit EOF during the probe is deferred and surfaced as io.EOF once any
// replayed bytes have been handed out.
func (b *bomStripReader) probe() {
	b.checked = true

	var head [len(utf8BOM)]byte

	n, err := io.ReadFull(b.r, head[:])
	b.pending = append(b.pending, head[:n]...)

	if n == len(utf8BOM) && bytes.Equal(head[:], utf8BOM[:]) {
		b.pending = b.pending[:0] // drop the BOM
	}

	switch {
	case err == nil:
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		b.err = io.EOF
	default:
		b.err = err
	}
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
