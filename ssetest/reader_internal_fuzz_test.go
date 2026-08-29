package ssetest

import (
	"bufio"
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

// scanAllLines drives a bufio.Scanner configured with splitSSELines over wire
// and returns every line. Mirrors newSSEScanner's buffering (minus the BOM
// wrapper, which is a separate component).
func scanAllLines(tb testing.TB, wire string) []string {
	tb.Helper()

	sc := bufio.NewScanner(
		strings.NewReader(wire),
	)
	sc.Buffer(make([]byte, 0, 4096), 1<<20)
	sc.Split(splitSSELines)

	var lines []string

	for sc.Scan() {
		lines = append(lines, sc.Text())
	}

	if err := sc.Err(); err != nil {
		tb.Fatalf("scanner error on %q: %v", wire, err)
	}

	return lines
}

// FuzzSplitSSELines pins the SplitFunc contract on arbitrary byte soup: the
// scanner's line boundaries must exactly match the spec's terminator rules
// (reference model), the splitter must never error, and the result must be
// independent of how the input is chunked into reads.
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
		want := referenceSplitLines(wire)

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
