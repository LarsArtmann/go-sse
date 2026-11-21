package sse

import (
	"io"
	"strings"
)

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
// Keys containing CR or LF are likewise out of contract: keys are protocol
// tokens ("elements", "signals"), and a multi-line key cannot prefix every
// output line.
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

// forEachLine walks s line by line, yielding each line without its
// terminator. CR (\r), LF (\n), and CRLF (\r\n) all terminate a line per the
// SSE spec (§ 9.2.5 end-of-line). A trailing terminator does not yield a
// final empty line, and the empty string yields exactly one empty line —
// the same semantics [splitLines] collects into a slice, without allocating
// one. The yield callback must not retain line or be called re-entrantly.
func forEachLine(s string, yield func(line string)) {
	if s == "" {
		yield("")

		return
	}

	start := 0

	for i := 0; i < len(s); {
		switch s[i] {
		case '\r':
			yield(s[start:i])
			i++

			if i < len(s) && s[i] == '\n' {
				i++ // CRLF: consume both characters as one line break
			}

			start = i
		case '\n':
			yield(s[start:i])
			i++
			start = i
		default:
			i++
		}
	}

	if start < len(s) {
		yield(s[start:])
	}
}

// splitLines splits a string into lines for SSE data field formatting.
// Each line in the SSE spec must be prefixed with "data: ".
// Per the SSE spec, CR (\r), LF (\n), and CRLF (\r\n) are all valid line endings.
// The no-CR/LF fast path allocates a single-element backing array; use
// [forEachLine] on hot paths that only consume the lines.
func splitLines(s string) []string {
	if !strings.ContainsAny(s, "\n\r") {
		return []string{s}
	}

	var lines []string

	forEachLine(s, func(line string) { lines = append(lines, line) })

	return lines
}
