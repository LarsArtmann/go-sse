package sse_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse"
)

func TestJoinLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{"single line", []string{"hello"}, "hello"},
		{"two lines", []string{"a", "b"}, "a\nb"},
		{"three lines", []string{"a", "b", "c"}, "a\nb\nc"},
		{"empty result", []string{}, ""},
		{"empty line", []string{""}, ""},
		{"with empty strings", []string{"a", "", "b"}, "a\n\nb"},
		{
			"datastar pattern",
			[]string{"selector #feed", "mode inner", "elements <div>"},
			"selector #feed\nmode inner\nelements <div>",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := sse.JoinLines(test.lines...)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestJoinLines_ProducesCorrectWireFormat(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	html := "<div>\n  <span>hi</span>\n</div>"

	err := sse.WriteEvent(&buf, sse.Event{
		Event: "datastar-patch-elements",
		Data:  sse.JoinLines("selector #feed", "mode inner", sse.KeyedLines("elements", html)),
	})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	want := "event: datastar-patch-elements\n" +
		"data: selector #feed\n" +
		"data: mode inner\n" +
		"data: elements <div>\n" +
		"data: elements   <span>hi</span>\n" +
		"data: elements </div>\n\n"

	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestKeyedLines_SingleLine(t *testing.T) {
	t.Parallel()

	got := sse.KeyedLines("selector", "#feed")
	want := "selector #feed"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestKeyedLines_MultiLine(t *testing.T) {
	t.Parallel()

	got := sse.KeyedLines("elements", "<div>\n  <span>hi</span>\n</div>")
	want := "elements <div>\nelements   <span>hi</span>\nelements </div>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestKeyedLines_EmptyValue(t *testing.T) {
	t.Parallel()

	got := sse.KeyedLines("elements", "")
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestKeyedLines_EmptyKey(t *testing.T) {
	t.Parallel()

	// Empty key prefixes each line with just a space — this is the literal
	// behavior (no key to write, only the separator). It is documented rather
	// than special-cased because an empty key is a caller bug, not a valid
	// keyed-data-line pattern. The test pins the behavior so it never changes
	// accidentally.
	got := sse.KeyedLines("", "hello\nworld")
	want := " hello\n world"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestKeyedLines_CRLFInValue(t *testing.T) {
	t.Parallel()

	got := sse.KeyedLines("elements", "<div>\r\n</div>")
	want := "elements <div>\nelements </div>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestKeyedLines_ProducesCorrectWireFormat(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	html := "<div id=\"feed\">\n  <span>1</span>\n</div>"

	err := sse.WriteEvent(&buf, sse.Event{
		Event: "datastar-patch-elements",
		Data:  "selector #feed\nmode inner\n" + sse.KeyedLines("elements", html),
	})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	want := "event: datastar-patch-elements\n" +
		"data: selector #feed\n" +
		"data: mode inner\n" +
		"data: elements <div id=\"feed\">\n" +
		"data: elements   <span>1</span>\n" +
		"data: elements </div>\n" +
		"\n"

	if buf.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestWriteKeyedLines_SingleLine(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteKeyedLines(&buf, "datastar-patch-signals", "signals", `{"progress":50}`)
	if err != nil {
		t.Fatalf("WriteKeyedLines: %v", err)
	}

	want := "event: datastar-patch-signals\n" +
		"data: signals {\"progress\":50}\n" +
		"\n"

	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteKeyedLines_MultiLine(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteKeyedLines(&buf, "datastar-patch-elements", "elements", "<div>\n</div>")
	if err != nil {
		t.Fatalf("WriteKeyedLines: %v", err)
	}

	want := "event: datastar-patch-elements\n" +
		"data: elements <div>\n" +
		"data: elements </div>\n" +
		"\n"

	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteKeyedLines_EmptyValue(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteKeyedLines(&buf, "test", "key", "")
	if err != nil {
		t.Fatalf("WriteKeyedLines: %v", err)
	}

	// Empty value → KeyedLines returns "" → WriteEvent emits a bare "data: \n".
	want := "event: test\ndata: \n\n"

	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteKeyedLines_CRLFInValue(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteKeyedLines(&buf, "evt", "k", "a\r\nb")
	if err != nil {
		t.Fatalf("WriteKeyedLines: %v", err)
	}

	// CRLF is normalized to LF by splitLines inside KeyedLines.
	want := "event: evt\ndata: k a\ndata: k b\n\n"

	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteKeyedLines_WriteError(t *testing.T) {
	t.Parallel()

	err := sse.WriteKeyedLines(&errorWriter{}, "evt", "k", "v")
	if err == nil {
		t.Fatal("expected error from errorWriter")
	}
}

func BenchmarkKeyedLines(b *testing.B) {
	singleLine := "<div>hello</div>"

	var multiLineBuilder strings.Builder
	multiLineBuilder.WriteString("<div id=\"feed\">\n")
	for range 100 {
		multiLineBuilder.WriteString(
			"  <article>\n    <h3>Title</h3>\n    <p>Body text</p>\n  </article>\n",
		)
	}
	multiLineBuilder.WriteString("</div>")
	multiLineHTML := multiLineBuilder.String()

	b.Run("single_line", func(b *testing.B) {
		b.ReportAllocs()

		for range b.N {
			_ = sse.KeyedLines("elements", singleLine)
		}
	})

	b.Run("multi_line_100", func(b *testing.B) {
		b.ReportAllocs()

		for range b.N {
			_ = sse.KeyedLines("elements", multiLineHTML)
		}
	})
}

func BenchmarkWriteEvent(b *testing.B) {
	simple := sse.Event{Event: "update", Data: "hello"}

	articleHTML := strings.Repeat("<article>\n  <h3>Title</h3>\n  <p>Body</p>\n</article>\n", 50)
	multiLine := sse.Event{
		Event: "datastar-patch-elements",
		Data:  "selector #feed\nmode inner\n" + sse.KeyedLines("elements", articleHTML),
		ID:    sse.NewEventID("evt-42"),
		Retry: 5000,
	}

	cases := []struct {
		name string
		evt  sse.Event
	}{
		{"simple", simple},
		{"multiline_50", multiLine},
	}

	for _, benchCase := range cases {
		b.Run(benchCase.name, func(b *testing.B) {
			var buf bytes.Buffer
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				buf.Reset()
				if err := sse.WriteEvent(&buf, benchCase.evt); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
