package sseparse_test

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse/sseparse"
)

func TestStreamReader_SequentialReads(t *testing.T) {
	t.Parallel()

	const wire = "event: connected\ndata: connected\n\n" +
		"event: sync:ack\ndata: {\"commandId\":\"42\",\"status\":\"confirmed\"}\n\n" +
		"event: report-updated\ndata: 2026-08-22\n\n"

	reader := sseparse.NewStreamReader(strings.NewReader(wire))

	evt1, err := reader.Next()
	if err != nil {
		t.Fatalf("first Next: %v", err)
	}

	if evt1.Type != "connected" {
		t.Errorf("event[0].Type: got %q, want %q", evt1.Type, "connected")
	}

	if evt1.Data() != "connected" {
		t.Errorf("event[0].Data: got %q, want %q", evt1.Data(), "connected")
	}

	evt2, err := reader.Next()
	if err != nil {
		t.Fatalf("second Next: %v", err)
	}

	if evt2.Type != "sync:ack" {
		t.Errorf("event[1].Type: got %q, want %q", evt2.Type, "sync:ack")
	}

	evt3, err := reader.Next()
	if err != nil {
		t.Fatalf("third Next: %v", err)
	}

	if evt3.Type != "report-updated" {
		t.Errorf("event[2].Type: got %q, want %q", evt3.Type, "report-updated")
	}

	_, err = reader.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("after all events: got %v, want io.EOF", err)
	}
}

func TestStreamReader_PreservesBufferedData(t *testing.T) {
	t.Parallel()

	const wire = "data: 1\n\ndata: 2\n\ndata: 3\n\ndata: 4\n\n"

	reader := sseparse.NewStreamReader(strings.NewReader(wire))

	for i := 1; i <= 4; i++ {
		evt, err := reader.Next()
		if err != nil {
			t.Fatalf("Next #%d: %v", i, err)
		}

		want := strconv.Itoa(i)
		if got := evt.Data(); got != want {
			t.Errorf("event #%d: got %q, want %q", i, got, want)
		}
	}

	_, err := reader.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("after all events: got %v, want io.EOF", err)
	}
}

func TestStreamReader_DatalessFramesSkipped(t *testing.T) {
	t.Parallel()

	const wire = ": heartbeat\n\n" +
		"id: 7\n\n" +
		"event: real\ndata: payload\n\n"

	reader := sseparse.NewStreamReader(strings.NewReader(wire))

	evt, err := reader.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}

	if evt.Type != "real" {
		t.Errorf("type: got %q, want %q", evt.Type, "real")
	}

	if evt.ID != "7" {
		t.Errorf("id (sticky): got %q, want %q", evt.ID, "7")
	}

	_, err = reader.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("after one event: got %v, want io.EOF", err)
	}
}

func TestStreamReader_FailingReader(t *testing.T) {
	t.Parallel()

	reader := sseparse.NewStreamReader(failingReader{})

	_, err := reader.Next()
	if err == nil {
		t.Fatal("expected error from StreamReader with failing reader")
	}

	if !errors.Is(err, errTestReadFailure) {
		t.Errorf("scan failure should wrap the underlying reader error: got %v", err)
	}
}

func TestMustReadNextEvent(t *testing.T) {
	t.Parallel()

	reader := sseparse.NewStreamReader(strings.NewReader("event: ping\ndata: pong\n\n"))

	evt := sseparse.MustReadNextEvent(t, reader)
	if evt.Type != "ping" {
		t.Errorf("type: got %q, want %q", evt.Type, "ping")
	}

	if evt.Data() != "pong" {
		t.Errorf("data: got %q, want %q", evt.Data(), "pong")
	}
}

// TestMustReadNextEvent_Failure pins the must-helper's fatal path: a failing
// reader surfaces as a tb.Fatalf (via recordingTB, which also proves the
// helper works with any testing.TB, not just *testing.T).
func TestMustReadNextEvent_Failure(t *testing.T) {
	t.Parallel()

	tb := &recordingTB{}

	evt := sseparse.MustReadNextEvent(tb, sseparse.NewStreamReader(failingReader{}))

	if len(tb.fatals) != 1 || !strings.Contains(tb.fatals[0], "read SSE event") {
		t.Errorf("expected read-failure fatal; got %v (event %+v)", tb.fatals, evt)
	}
}
