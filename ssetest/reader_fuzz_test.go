package ssetest_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-sse/ssetest"
)

// FuzzReadEvents asserts that the SSE wire parser never panics on arbitrary
// input, and that well-formed input round-trips into events.
func FuzzReadEvents(f *testing.F) {
	seeds := []string{
		"",
		"\n",
		"data: hello\n\n",
		"event: feed\ndata: line\ndata: line\ndata: line\n\n",
		"event: feed\r\nid: 42\r\nretry: 3000\r\ndata: crlf\r\n\r\n",
		": comment only\ndata: x\n\n",
		": heartbeat\n\n",
		"id: 1\n\nid: 2\n\n",
		"retry: 3000\n\n",
		"event: named\nid: 9\nretry: 100\n\n",
		"retry: not-a-number\ndata: x\n\n",
		"data: no trailing newline",
		"fieldwithoutcolon\n\n",
		"data:nospace\n\n",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, wire string) {
		events, err := ssetest.ReadEvents(strings.NewReader(wire))
		if err != nil {
			t.Fatalf("ReadEvents should never fail on an in-memory reader: %v", err)
		}

		// Dataless-frame invariant: every dispatched event carries at least one
		// data line. Comment/id/retry-only frames must never surface as events.
		for _, evt := range events {
			if len(evt.DataLines) == 0 {
				t.Errorf("event dispatched without data lines: %s", evt)
			}

			_ = evt.String()
			_ = evt.Data()
		}

		// Exact well-formed input must parse to exactly one event. (Substring
		// matching would be wrong: "0data: hello\n\n" contains the substring
		// but is a different field name, so it correctly dispatches nothing.)
		if wire == "data: hello\n\n" && len(events) != 1 {
			t.Errorf("exact well-formed input parsed to %d events, want 1", len(events))
		}
	})
}
