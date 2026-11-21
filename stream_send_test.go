package sse_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-sse"
)

func TestStream_Send(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	err := stream.Send(sse.Event{Event: "update", Data: "<div>new</div>"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	want := "event: update\ndata: <div>new</div>\n\n"
	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
	}
}

func TestStream_SendMultiple(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	_ = stream.Send(sse.Event{Event: "e1", Data: "d1"})
	_ = stream.Send(sse.Event{Event: "e2", Data: "d2"})

	body := w.Body.String()
	if !strings.Contains(body, "event: e1\ndata: d1\n\n") {
		t.Errorf("missing e1: %q", body)
	}

	if !strings.Contains(body, "event: e2\ndata: d2\n\n") {
		t.Errorf("missing e2: %q", body)
	}
}

func TestStream_SendData(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	err := stream.SendData("update", "<ul><li>item</li></ul>")
	if err != nil {
		t.Fatalf("SendData: %v", err)
	}

	want := "event: update\ndata: <ul><li>item</li></ul>\n\n"
	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
	}
}

func TestStream_SendLines(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	html := "<div>\n  <span>hi</span>\n</div>"

	err := stream.SendLines(
		"datastar-patch-elements",
		"selector #feed",
		"mode inner",
		sse.KeyedLines("elements", html),
	)
	if err != nil {
		t.Fatalf("SendLines: %v", err)
	}

	want := "event: datastar-patch-elements\n" +
		"data: selector #feed\n" +
		"data: mode inner\n" +
		"data: elements <div>\n" +
		"data: elements   <span>hi</span>\n" +
		"data: elements </div>\n" +
		"\n"

	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
	}
}

func TestStream_SendKeyed(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	err := stream.SendKeyed("datastar-patch-signals", "signals", `{"progress":50}`)
	if err != nil {
		t.Fatalf("SendKeyed: %v", err)
	}

	want := "event: datastar-patch-signals\n" +
		"data: signals {\"progress\":50}\n" +
		"\n"

	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
	}
}

func TestStream_SendKeyed_MultiLine(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	err := stream.SendKeyed("datastar-patch-elements", "elements", "<div>\n</div>")
	if err != nil {
		t.Fatalf("SendKeyed: %v", err)
	}

	want := "event: datastar-patch-elements\n" +
		"data: elements <div>\n" +
		"data: elements </div>\n" +
		"\n"

	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
	}
}
