package sseparse_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse/sseparse"
)

// recordingTB captures Fatal/Errorf calls instead of failing the test run.
// It satisfies testing.TB by embedding the interface; methods the Require*
// helpers do not call are never reached. All helpers accept testing.TB, so
// this also proves Ginkgo's GinkgoT() and *testing.B work with them.
type recordingTB struct {
	testing.TB

	fatals []string
	errors []string
}

func (r *recordingTB) Helper() {}

func (r *recordingTB) Fatal(args ...any) {
	r.fatals = append(r.fatals, fmt.Sprint(args...))
}

func (r *recordingTB) Fatalf(format string, args ...any) {
	r.fatals = append(r.fatals, fmt.Sprintf(format, args...))
}

func (r *recordingTB) Error(args ...any) {
	r.errors = append(r.errors, fmt.Sprint(args...))
}

func (r *recordingTB) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func TestRequireEventCount_Failure(t *testing.T) {
	t.Parallel()

	tb := &recordingTB{}
	sseparse.RequireEventCount(tb, nil, 2)

	if len(tb.fatals) != 1 || !strings.Contains(tb.fatals[0], "got 0, want 2") {
		t.Errorf("expected count mismatch fatal; got %v", tb.fatals)
	}
}

func TestRequireEventType_Failure(t *testing.T) {
	t.Parallel()

	tb := &recordingTB{}
	sseparse.RequireEventType(tb, sseparse.Event{Type: "feed"}, "alert")

	if len(tb.fatals) != 1 || !strings.Contains(tb.fatals[0], `"alert"`) {
		t.Errorf("expected type mismatch fatal; got %v", tb.fatals)
	}
}

func TestRequireData_Failure(t *testing.T) {
	t.Parallel()

	tb := &recordingTB{}
	sseparse.RequireData(tb, sseparse.Event{DataLines: []string{"a"}}, "b")

	if len(tb.errors) != 1 || !strings.Contains(tb.errors[0], "data") {
		t.Errorf("expected data mismatch error; got %v", tb.errors)
	}
}

func TestRequireDataContains_Failure(t *testing.T) {
	t.Parallel()

	tb := &recordingTB{}
	sseparse.RequireDataContains(tb, sseparse.Event{DataLines: []string{"hello"}}, "goodbye")

	if len(tb.errors) != 1 || !strings.Contains(tb.errors[0], "goodbye") {
		t.Errorf("expected substring error; got %v", tb.errors)
	}
}

func TestRequireEventID_Failure(t *testing.T) {
	t.Parallel()

	tb := &recordingTB{}
	sseparse.RequireEventID(tb, sseparse.Event{ID: "41"}, "42")

	if len(tb.fatals) != 1 || !strings.Contains(tb.fatals[0], `"41"`) {
		t.Errorf("expected ID mismatch fatal; got %v", tb.fatals)
	}
}

func TestRequireRetry_Failure(t *testing.T) {
	t.Parallel()

	tb := &recordingTB{}
	sseparse.RequireRetry(tb, sseparse.Event{Retry: 1000}, 3000)

	if len(tb.fatals) != 1 || !strings.Contains(tb.fatals[0], "1000") {
		t.Errorf("expected retry mismatch fatal; got %v", tb.fatals)
	}
}

func TestMustReadEvents_FailingReader(t *testing.T) {
	t.Parallel()

	tb := &recordingTB{}
	sseparse.MustReadEvents(tb, failingReader{})

	if len(tb.fatals) != 1 {
		t.Errorf("expected fatal from failing reader; got %v", tb.fatals)
	}
}

func TestMustReadNEvents(t *testing.T) {
	t.Parallel()

	const wire = "data: 1\n\ndata: 2\n\n"

	events := sseparse.MustReadNEvents(t, strings.NewReader(wire), 2)
	sseparse.RequireEventCount(t, events, 2)

	tb := &recordingTB{}
	sseparse.MustReadNEvents(tb, failingReader{}, 2)

	if len(tb.fatals) != 1 {
		t.Errorf("expected fatal from failing reader; got %v", tb.fatals)
	}
}

func TestFindByType(t *testing.T) {
	t.Parallel()

	events := []sseparse.Event{
		{Type: "feed", DataLines: []string{"1"}},
		{Type: "alert", DataLines: []string{"2"}},
	}

	evt, ok := sseparse.FindByType(events, "alert")
	if !ok || evt.Data() != "2" {
		t.Errorf("FindByType(alert): got (%v, %v)", evt, ok)
	}

	if _, ok := sseparse.FindByType(events, "missing"); ok {
		t.Error("FindByType(missing) should not be found")
	}
}

func TestFilterByType(t *testing.T) {
	t.Parallel()

	events := []sseparse.Event{
		{Type: "feed"},
		{Type: "alert"},
		{Type: "feed"},
	}

	feed := sseparse.FilterByType(events, "feed")
	if len(feed) != 2 {
		t.Errorf("FilterByType(feed): got %d, want 2", len(feed))
	}

	if len(events) != 3 {
		t.Errorf("FilterByType must not mutate the input: got %d, want 3", len(events))
	}
}

func TestEvent_String(t *testing.T) {
	t.Parallel()

	full := sseparse.Event{Type: "feed", ID: "1", Retry: 2, DataLines: []string{"a", "b"}}
	if got := full.String(); got != "Event{type=feed id=1 retry=2 datalines=2}" {
		t.Errorf("String(): got %q", got)
	}

	plain := sseparse.Event{Type: "feed"}
	if got := plain.String(); got != "Event{type=feed datalines=0}" {
		t.Errorf("String(): got %q", got)
	}

	if got := sseparse.EventsString(nil); got != "(no events)" {
		t.Errorf("EventsString(nil): got %q", got)
	}
}

// TestHelpers_AcceptTestingB proves the helpers work with *testing.B, not just
// *testing.T — the reason every public helper takes testing.TB.
func TestHelpers_AcceptTestingB(t *testing.T) {
	t.Parallel()

	b := &testing.B{}
	sseparse.RequireEventCount(b, []sseparse.Event{{Type: "x"}}, 1)
	sseparse.RequireEventType(b, sseparse.Event{Type: "x"}, "x")
	sseparse.RequireEventID(b, sseparse.Event{ID: "1"}, "1")
	sseparse.RequireRetry(b, sseparse.Event{}, 0)

	if b.Failed() {
		t.Error("helper calls on *testing.B should not fail")
	}
}

// progressSignal is the typed want for the RequireDataJSON tests.
type progressSignal struct {
	Progress float64 `json:"progress"`
	Name     string  `json:"name"`
}

func TestRequireDataJSON_TypedWant(t *testing.T) {
	t.Parallel()

	evt := sseparse.Event{DataLines: []string{`{"progress":50,"name":"upload"}`}}
	sseparse.RequireDataJSON(t, evt, progressSignal{Progress: 50, Name: "upload"})
}

func TestRequireDataJSON_MapWant(t *testing.T) {
	t.Parallel()

	evt := sseparse.Event{DataLines: []string{`{"progress":50}`}}
	sseparse.RequireDataJSON(t, evt, map[string]any{"progress": 50.0})
}

func TestRequireDataJSON_KeyOrderAndWhitespaceDoNotMatter(t *testing.T) {
	t.Parallel()

	evt := sseparse.Event{DataLines: []string{`{ "name" : "upload" ,  "progress":50 }`}}
	sseparse.RequireDataJSON(t, evt, progressSignal{Progress: 50, Name: "upload"})
}

func TestRequireDataJSON_MismatchFails(t *testing.T) {
	t.Parallel()

	tb := &recordingTB{}
	evt := sseparse.Event{DataLines: []string{`{"progress":40}`}}
	sseparse.RequireDataJSON(tb, evt, map[string]any{"progress": 50.0})

	if len(tb.errors) != 1 {
		t.Fatalf("expected 1 error, got %v", tb.errors)
	}
}

func TestRequireDataJSON_InvalidJSONFailsFatally(t *testing.T) {
	t.Parallel()

	tb := &recordingTB{}
	evt := sseparse.Event{DataLines: []string{`{not json`}}
	sseparse.RequireDataJSON(tb, evt, map[string]any{})

	if len(tb.fatals) != 1 || !strings.Contains(tb.fatals[0], "{not json") {
		t.Fatalf("expected fatal naming the payload; got %v", tb.fatals)
	}
}

func TestRequireDataJSON_AcceptsTestingB(t *testing.T) {
	t.Parallel()

	b := &testing.B{}
	sseparse.RequireDataJSON(b, sseparse.Event{DataLines: []string{`"x"`}}, "x")

	if b.Failed() {
		t.Error("RequireDataJSON on *testing.B should not fail")
	}
}
