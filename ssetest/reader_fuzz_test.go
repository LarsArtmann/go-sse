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
		"id: 1\n\nid: 2\n\n",
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

		if strings.Contains(wire, "data: hello\n\n") && len(events) == 0 {
			t.Error("well-formed seed event was not parsed")
		}

		for _, evt := range events {
			_ = evt.String()
			_ = evt.Data()
		}
	})
}
