package sse_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse"
)

func TestWriteEvent_NamedEventWithData(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteEvent(&buf, sse.Event{
		Event: "todoCreated",
		Data:  "<div>Buy milk</div>",
	})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	want := "event: todoCreated\ndata: <div>Buy milk</div>\n\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteEvent_UnnamedMessageEvent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteEvent(&buf, sse.Event{Data: "<div>content</div>"})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	want := "data: <div>content</div>\n\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteEvent_MultiLineData(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteEvent(&buf, sse.Event{
		Event: "update",
		Data:  "line1\nline2\nline3",
	})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	want := "event: update\ndata: line1\ndata: line2\ndata: line3\n\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteEvent_EventID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteEvent(&buf, sse.Event{
		Data: "payload",
		ID:   sse.NewEventID("42"),
	})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("id: 42\n")) {
		t.Errorf("missing id: 42 in output %q", buf.String())
	}
}

func TestWriteEvent_Retry(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteEvent(&buf, sse.Event{
		Data:  "payload",
		Retry: 5000,
	})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("retry: 5000\n")) {
		t.Errorf("missing retry: 5000 in output %q", buf.String())
	}
}

func TestWriteEvent_CompleteEvent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteEvent(&buf, sse.Event{
		Event: "fullEvent",
		Data:  "multi\nline\ndata",
		ID:    sse.NewEventID("evt-123"),
		Retry: 3000,
	})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	output := buf.String()
	if output[:1] != "e" {
		t.Errorf("expected event: prefix")
	}

	if !bytes.Contains(buf.Bytes(), []byte("event: fullEvent\n")) {
		t.Errorf("missing event line")
	}

	if !bytes.Contains(buf.Bytes(), []byte("data: multi\ndata: line\ndata: data\n")) {
		t.Errorf("missing multi-line data")
	}

	if !bytes.Contains(buf.Bytes(), []byte("id: evt-123\n")) {
		t.Errorf("missing id line")
	}

	if !bytes.Contains(buf.Bytes(), []byte("retry: 3000\n")) {
		t.Errorf("missing retry line")
	}

	if output[len(output)-2:] != "\n\n" {
		t.Errorf("expected blank line terminator")
	}
}

func TestWriteEvent_EmptyData(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteEvent(&buf, sse.Event{Event: "empty"})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	want := "event: empty\ndata: \n\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteEvent_CRLFInData(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteEvent(&buf, sse.Event{Data: "line1\r\nline2"})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	want := "data: line1\ndata: line2\n\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteEvent_TrailingNewline(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteEvent(&buf, sse.Event{Event: "trailing", Data: "line1\n"})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	want := "event: trailing\ndata: line1\n\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteEvent_WriteError(t *testing.T) {
	t.Parallel()

	err := sse.WriteEvent(&errorWriter{}, sse.Event{Event: "fail"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteHeartbeat(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteHeartbeat(&buf)
	if err != nil {
		t.Fatalf("WriteHeartbeat: %v", err)
	}

	want := ": heartbeat\n\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestParseEventID_Valid(t *testing.T) {
	t.Parallel()

	cases := []string{"", "42", "evt-99", "550e8400-e29b-41d4-a716-446655440000"}

	for _, input := range cases {
		id, err := sse.ParseEventID(input)
		if err != nil {
			t.Errorf("ParseEventID(%q): unexpected error: %v", input, err)
		}

		if id.Get() != input {
			t.Errorf("ParseEventID(%q): got %q", input, id.Get())
		}
	}
}

func TestParseEventID_RejectsNewlines(t *testing.T) {
	t.Parallel()

	cases := []string{"evt\n42", "evt\r42", "evt\r\n42", "\n"}

	for _, input := range cases {
		_, err := sse.ParseEventID(input)
		if err == nil {
			t.Errorf("ParseEventID(%q): expected error", input)
		}
	}
}

func TestNewEventID(t *testing.T) {
	t.Parallel()

	id := sse.NewEventID("evt-1")
	if id.Get() != "evt-1" {
		t.Errorf("got %q", id.Get())
	}
}

func TestEventID_IsZero(t *testing.T) {
	t.Parallel()

	if !sse.NewEventID("").IsZero() {
		t.Error("empty EventID should be zero")
	}

	if sse.NewEventID("x").IsZero() {
		t.Error("non-empty EventID should not be zero")
	}
}

func TestMustParseEventID_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()

	sse.MustParseEventID("bad\nvalue")
}

func TestMustParseEventID_Valid(t *testing.T) {
	t.Parallel()

	id := sse.MustParseEventID("evt-42")
	if id.Get() != "evt-42" {
		t.Errorf("got %q, want evt-42", id.Get())
	}
}

func TestWriteRetry(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteRetry(&buf, 3000)
	if err != nil {
		t.Fatalf("WriteRetry: %v", err)
	}

	want := "retry: 3000\n\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteRetry_WriteError(t *testing.T) {
	t.Parallel()

	err := sse.WriteRetry(&errorWriter{}, 3000)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteEvent_LoneCarriageReturn(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteEvent(&buf, sse.Event{Data: "line1\rline2"})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	want := "data: line1\ndata: line2\n\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteEvent_NegativeRetryIsUint(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	// Retry is uint; the > 0 guard means zero-value emits nothing.
	err := sse.WriteEvent(&buf, sse.Event{Data: "x"})
	if err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	if strings.Contains(buf.String(), "retry:") {
		t.Errorf("zero Retry should not emit retry field: %q", buf.String())
	}
}

func TestParseEventID_Unicode(t *testing.T) {
	t.Parallel()

	cases := []string{
		"\u65e5\u672c\u8a9e",
		"\U0001f31e",
		"caf\u00e9-m\u00fcnchen-123",
	}

	for _, input := range cases {
		id, err := sse.ParseEventID(input)
		if err != nil {
			t.Errorf("ParseEventID(%q): unexpected error: %v", input, err)
		}

		if id.Get() != input {
			t.Errorf("ParseEventID(%q): got %q", input, id.Get())
		}
	}
}

type errorWriter struct{}

func (e *errorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write error")
}

var _ io.Writer = (*errorWriter)(nil)

func TestEvent_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		evt  sse.Event
		want string
	}{
		{
			name: "all fields",
			evt: sse.Event{
				Event: "update",
				Data:  "<div>new</div>",
				ID:    sse.NewEventID("42"),
				Retry: 3000,
			},
			want: "{event:update id:42 retry:3000 data:<div>new</div>}",
		},
		{
			name: "data only",
			evt:  sse.Event{Data: "hello"},
			want: "{data:hello}",
		},
		{
			name: "event name only",
			evt:  sse.Event{Event: "ping"},
			want: "{event:ping}",
		},
		{
			name: "empty event",
			evt:  sse.Event{},
			want: "{}",
		},
		{
			name: "id and retry only",
			evt:  sse.Event{ID: sse.NewEventID("7"), Retry: 500},
			want: "{id:7 retry:500}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.evt.String(); got != tc.want {
				t.Errorf("String(): got %q, want %q", got, tc.want)
			}
		})
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
		Data: "selector #feed\nmode inner\n" + sse.KeyedLines("elements", html),
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
