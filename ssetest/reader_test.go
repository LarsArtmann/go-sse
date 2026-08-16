package ssetest_test

import (
	"errors"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-sse/ssetest"
)

var errTestReadFailure = errors.New("test: simulated read failure")

type failingReader struct{}

func (failingReader) Read(p []byte) (int, error) {
	return 0, errTestReadFailure
}

func TestReadEvents_FullWireFormat(t *testing.T) {
	t.Parallel()

	const wire = ": comment lines are ignored\r\n" +
		"event: feed\n" +
		"id: 42\n" +
		"retry: 3000\n" +
		"data: first line\n" +
		"data: second line\n" +
		"\n" +
		"data: unnamed event\n" +
		"\n"

	events, err := ssetest.ReadEvents(strings.NewReader(wire))
	if err != nil {
		t.Fatalf("read events: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("event count: got %d, want 2", len(events))
	}

	named := events[0]
	if named.Type != "feed" {
		t.Errorf("type: got %q, want %q", named.Type, "feed")
	}

	if named.ID != "42" {
		t.Errorf("id: got %q, want %q", named.ID, "42")
	}

	if named.Retry != 3000 {
		t.Errorf("retry: got %d, want 3000", named.Retry)
	}

	if got := named.Data(); got != "first line\nsecond line" {
		t.Errorf("data: got %q, want two joined lines", got)
	}

	if len(named.DataLines) != 2 {
		t.Errorf("datalines: got %d, want 2", len(named.DataLines))
	}

	unnamed := events[1]
	if unnamed.Type != "" {
		t.Errorf("unnamed event type: got %q, want empty", unnamed.Type)
	}

	if got := unnamed.Data(); got != "unnamed event" {
		t.Errorf("unnamed event data: got %q", got)
	}
}

func TestReadEvents_CRLFLineEndings(t *testing.T) {
	t.Parallel()

	const wire = "event: feed\r\n" +
		"id: 7\r\n" +
		"data: hello\r\n" +
		"\r\n"

	events, err := ssetest.ReadEvents(strings.NewReader(wire))
	if err != nil {
		t.Fatalf("read events: %v", err)
	}

	ssetest.RequireEventCount(t, events, 1)
	ssetest.RequireEventType(t, events[0], "feed")
	ssetest.RequireEventID(t, events[0], "7")
	ssetest.RequireData(t, events[0], "hello")
}

// TestReadEvents_IncompleteFinalFrameDiscarded pins spec § 9.2.6: "Once the
// end of the file is reached, any pending data must be discarded." A frame
// without a blank line before EOF never dispatches — this is also pinned by
// the WPT format-data-before-final-empty-line vector. (Before the spec
// conformance fixes, this test asserted the opposite, lenient behavior.)
func TestReadEvents_IncompleteFinalFrameDiscarded(t *testing.T) {
	t.Parallel()

	events, err := ssetest.ReadEvents(strings.NewReader("data: tail\n"))
	if err != nil {
		t.Fatalf("read events: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("incomplete final frame: got %d events, want 0", len(events))
	}

	// The blank line is what dispatches: with it, the same frame surfaces.
	events, err = ssetest.ReadEvents(strings.NewReader("data: tail\n\n"))
	if err != nil {
		t.Fatalf("read events: %v", err)
	}

	ssetest.RequireEventCount(t, events, 1)
	ssetest.RequireData(t, events[0], "tail")
}

// TestReadEvents_DatalessFramesNeverDispatch guards the SSE-spec rule that
// only frames with data lines dispatch: comment frames (heartbeats) and
// id/retry-only frames must not surface as phantom empty events.
func TestReadEvents_DatalessFramesNeverDispatch(t *testing.T) {
	t.Parallel()

	const wire = ": heartbeat\n\n" +
		"id: 7\n\n" +
		"retry: 5000\n\n" +
		"event: feed\n" +
		"data: real\n\n" +
		": heartbeat\n\n"

	events, err := ssetest.ReadEvents(strings.NewReader(wire))
	if err != nil {
		t.Fatalf("read events: %v", err)
	}

	ssetest.RequireEventCount(t, events, 1)
	ssetest.RequireData(t, events[0], "real")
}

func TestReadEvents_Empty(t *testing.T) {
	t.Parallel()

	events, err := ssetest.ReadEvents(strings.NewReader(""))
	if err != nil {
		t.Fatalf("read events: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("empty input: got %d events, want 0", len(events))
	}
}

func TestReadEvents_InvalidRetryIgnored(t *testing.T) {
	t.Parallel()

	events, err := ssetest.ReadEvents(strings.NewReader("retry: not-a-number\ndata: x\n\n"))
	if err != nil {
		t.Fatalf("read events: %v", err)
	}

	ssetest.RequireEventCount(t, events, 1)
	ssetest.RequireRetry(t, events[0], 0)
}

func TestReadEvents_FailingReader_ClassifiedTransient(t *testing.T) {
	t.Parallel()

	events, err := ssetest.ReadEvents(failingReader{})
	if err == nil {
		t.Fatal("expected error from ReadEvents with failing reader")
	}

	if len(events) != 0 {
		t.Errorf("failing reader should return 0 events; got %d", len(events))
	}

	if errorfamily.Code(err) != ssetest.CodeSSEScanFailed {
		t.Errorf("error code: got %q, want %q", errorfamily.Code(err), ssetest.CodeSSEScanFailed)
	}

	if !errorfamily.IsRetryable(err) {
		t.Error("scan failure should classify as Transient")
	}
}

func TestReadNEvents_ReturnsBeforeEOF(t *testing.T) {
	t.Parallel()

	const wire = "data: 1\n\n" +
		"data: 2\n\n" +
		"data: 3\n\n"

	events, err := ssetest.ReadNEvents(strings.NewReader(wire), 2)
	if err != nil {
		t.Fatalf("read 2 events: %v", err)
	}

	ssetest.RequireEventCount(t, events, 2)

	if got := events[0].Data(); got != "1" {
		t.Errorf("event[0]: got %q, want %q", got, "1")
	}
}

func TestReadNEvents_ZeroOrNegative(t *testing.T) {
	t.Parallel()

	for _, count := range []int{0, -1} {
		events, err := ssetest.ReadNEvents(strings.NewReader("data: 1\n\n"), count)
		if err != nil {
			t.Fatalf("ReadNEvents(%d): %v", count, err)
		}

		if len(events) != 0 {
			t.Errorf("ReadNEvents(%d): got %d events, want 0", count, len(events))
		}
	}
}

func TestReadNEvents_FewerThanRequested(t *testing.T) {
	t.Parallel()

	events, err := ssetest.ReadNEvents(strings.NewReader("data: 1\n\ndata: 2\n\n"), 10)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}

	ssetest.RequireEventCount(t, events, 2)
}

func TestReadNEvents_ErrorAfterEventsIsCleanClose(t *testing.T) {
	t.Parallel()

	wire := &partialReader{valid: "data: 1\n\n"}

	events, err := ssetest.ReadNEvents(wire, 10)
	if err != nil {
		t.Fatalf("error after collected events should be a clean close: %v", err)
	}

	ssetest.RequireEventCount(t, events, 1)
}

func TestReadNEvents_FailingReader(t *testing.T) {
	t.Parallel()

	events, err := ssetest.ReadNEvents(failingReader{}, 10)
	if err == nil {
		t.Fatal("expected error from ReadNEvents with failing reader")
	}

	if len(events) != 0 {
		t.Errorf("failing reader: got %d events, want 0", len(events))
	}
}

// partialReader serves its valid SSE bytes across successive reads — like a
// network connection chunking the stream — then fails, simulating a connection
// drop mid-stream. It must tolerate multiple partial reads: the reader's BOM
// probe consumes the first bytes through its own read.
type partialReader struct {
	valid string
	off   int
}

func (p *partialReader) Read(b []byte) (int, error) {
	if p.off >= len(p.valid) {
		return 0, errTestReadFailure
	}

	n := copy(b, p.valid[p.off:])
	p.off += n

	return n, nil
}

func BenchmarkReadEvents(b *testing.B) {
	wire := strings.Repeat("event: feed\nid: 1\ndata: line one\ndata: line two\n\n", 1000)
	reader := strings.NewReader(wire)

	b.SetBytes(int64(len(wire)))
	b.ResetTimer()

	for b.Loop() {
		reader.Reset(wire)

		if _, err := ssetest.ReadEvents(reader); err != nil {
			b.Fatal(err)
		}
	}
}
