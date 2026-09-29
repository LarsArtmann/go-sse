package sseparse

import (
	"strings"
	"testing"
)

// referenceSplitLines is the spec § 9.2.5 line model the fuzz property checks
// against: split on CR, LF, or CRLF; a CRLF pair is one terminator; a final
// terminator produces no trailing empty line; an unterminated final segment
// is still a line. Independent implementation, same rules as splitSSELines.
func referenceSplitLines(s string) []string {
	var lines []string

	start := 0

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\n':
			lines = append(lines, s[start:i])
			start = i + 1
		case '\r':
			lines = append(lines, s[start:i])

			if i+1 < len(s) && s[i+1] == '\n' {
				i++ // CRLF: one terminator
			}

			start = i + 1
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}

// scanAllLines drives the production scanner pipeline (newSSEScanner: BOM
// strip, buffer setup, splitSSELines) over wire and returns every line.
// Consolidated onto newSSEScanner so the fuzz exercises the real buffer
// setup — including the min(initialLineCap, cap) sizing — instead of a
// hand-rolled copy that silently rots when newSSEScanner changes.
func scanAllLines(tb testing.TB, wire string) []string {
	tb.Helper()

	sc := newSSEScanner(strings.NewReader(wire), DefaultMaxLineBytes)

	var lines []string

	for sc.Scan() {
		lines = append(lines, sc.Text())
	}

	if err := sc.Err(); err != nil {
		tb.Fatalf("scanner error on %q: %v", wire, err)
	}

	return lines
}

// FuzzSplitSSELines pins the production line-splitting pipeline on arbitrary
// byte soup: strip-one-leading-BOM followed by the splitter must exactly
// match the spec's terminator rules (independent reference model), the
// scanner must never error, and the result must be independent of how the
// input is chunked into reads. (The BOM strip is part of newSSEScanner, so
// the reference model sees the same input the splitter does.)
func FuzzSplitSSELines(f *testing.F) {
	seeds := []string{
		"",
		"\n",
		"\r",
		"\r\n",
		"a",
		"a\n",
		"a\r",
		"a\r\n",
		"a\n\n",
		"a\r\r",
		"a\r\n\r\n",
		"a\nb",
		"a\rb\nc\r\n",
		"\r\n\r",
		"data: x\r\nid: 1\n",
		"\xEF\xBB\xBFdata: x\n",
		strings.Repeat("ab", 500) + "\r\n",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, wire string) {
		got := scanAllLines(t, wire)
		want := referenceSplitLines(strings.TrimPrefix(wire, string(utf8BOM[:])))

		if len(got) != len(want) {
			t.Fatalf("splitSSELines(%q): got %d lines %q, want %d %q",
				wire, len(got), got, len(want), want)
		}

		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("splitSSELines(%q): line[%d] got %q, want %q",
					wire, i, got[i], want[i])
			}
		}
	})
}
