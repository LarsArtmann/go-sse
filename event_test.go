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

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.evt.String(); got != test.want {
				t.Errorf("String(): got %q, want %q", got, test.want)
			}
		})
	}
}
