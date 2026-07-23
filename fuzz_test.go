package sse_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse"
)

func FuzzWriteEvent(f *testing.F) {
	f.Add("update", "<div>data</div>", "evt-1", uint(5000))
	f.Add("", "multi\nline", "", uint(0))
	f.Add("e", "line1\r\nline2\rline3", "id", uint(100))
	f.Add("named", "", "only-id", uint(0))
	f.Add("", "", "", uint(0))

	f.Fuzz(func(t *testing.T, event, data, id string, retry uint) {
		evt := sse.Event{
			Event: event,
			Data:  data,
			ID:    sse.NewEventID(id),
			Retry: retry,
		}

		var buf bytes.Buffer

		err := sse.WriteEvent(&buf, evt)
		if err != nil {
			t.Fatalf("WriteEvent: %v", err)
		}

		output := buf.String()

		if len(output) < 2 || output[len(output)-2:] != "\n\n" {
			t.Errorf("output must end with \\n\\n: %q", output)
		}

		if !strings.Contains(output, "data: ") {
			t.Errorf("output must contain at least one data: line: %q", output)
		}
	})
}

func FuzzParseEventID(f *testing.F) {
	f.Add("evt-42")
	f.Add("")
	f.Add("550e8400-e29b-41d4-a716-446655440000")
	f.Add("simple")

	f.Fuzz(func(t *testing.T, input string) {
		id, err := sse.ParseEventID(input)

		if strings.ContainsAny(input, "\n\r") {
			if err == nil {
				t.Errorf("ParseEventID(%q): expected error for newline-containing input", input)
			}

			return
		}

		if err != nil {
			t.Errorf("ParseEventID(%q): unexpected error: %v", input, err)
		}

		if id.Get() != input {
			t.Errorf("ParseEventID(%q): got %q", input, id.Get())
		}
	})
}
