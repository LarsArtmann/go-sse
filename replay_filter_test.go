package sse_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-sse"
)

// filteredMemoryStore implements FilteredEventStore by wrapping memoryStore
// and filtering in EventsAfterFiltered.
type filteredMemoryStore struct {
	events []sse.Event
}

func (m *filteredMemoryStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	if lastID.IsZero() {
		return m.events, nil
	}

	for i, evt := range m.events {
		if evt.ID.Get() == lastID.Get() {
			if i+1 < len(m.events) {
				return m.events[i+1:], nil
			}

			return nil, nil
		}
	}

	return nil, nil
}

func (m *filteredMemoryStore) EventsAfterFiltered(
	lastID sse.EventID, pred func(sse.Event) bool,
) ([]sse.Event, error) {
	all, err := m.EventsAfter(lastID)
	if err != nil {
		return nil, err
	}

	filtered := make([]sse.Event, 0, len(all))
	for _, evt := range all {
		if pred(evt) {
			filtered = append(filtered, evt)
		}
	}

	return filtered, nil
}

func testStore() *filteredMemoryStore {
	return &filteredMemoryStore{
		events: []sse.Event{
			{Event: "message", Data: "msg1", ID: sse.NewEventID("1")},
			{Event: "reaction", Data: "react1", ID: sse.NewEventID("2")},
			{Event: "message", Data: "msg2", ID: sse.NewEventID("3")},
			{Event: "reaction", Data: "react2", ID: sse.NewEventID("4")},
		},
	}
}

func TestReplayFiltered_FilteredEventStorePath(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	n, err := sse.ReplayFiltered(stream, testStore(), sse.NewEventID(""),
		func(evt sse.Event) bool { return evt.Event == "message" })
	if err != nil {
		t.Fatalf("ReplayFiltered: %v", err)
	}

	if n != 2 {
		t.Errorf("expected 2 messages, got %d", n)
	}

	body := w.Body.String()

	if !strings.Contains(body, "data: msg1") {
		t.Errorf("missing msg1 in %q", body)
	}

	if !strings.Contains(body, "data: msg2") {
		t.Errorf("missing msg2 in %q", body)
	}

	if strings.Contains(body, "data: react") {
		t.Errorf("should not contain reactions in %q", body)
	}
}

func TestReplayFiltered_FallbackPath(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	// Use plain memoryStore (does NOT implement FilteredEventStore)
	store := &memoryStore{
		events: []sse.Event{
			{Event: "message", Data: "msg1", ID: sse.NewEventID("1")},
			{Event: "reaction", Data: "react1", ID: sse.NewEventID("2")},
			{Event: "message", Data: "msg2", ID: sse.NewEventID("3")},
		},
	}

	n, err := sse.ReplayFiltered(stream, store, sse.NewEventID(""),
		func(evt sse.Event) bool { return evt.Event == "message" })
	if err != nil {
		t.Fatalf("ReplayFiltered fallback: %v", err)
	}

	if n != 2 {
		t.Errorf("expected 2 messages, got %d", n)
	}

	body := w.Body.String()

	if !strings.Contains(body, "data: msg1") {
		t.Errorf("missing msg1 in %q", body)
	}

	if strings.Contains(body, "data: react") {
		t.Errorf("should not contain reactions in %q", body)
	}
}

func TestReplayFiltered_NilPredDelegatesToReplay(t *testing.T) {
	t.Parallel()

	stream, _ := newTestStream(t)

	n, err := sse.ReplayFiltered(stream, testStore(), sse.NewEventID(""), nil)
	if err != nil {
		t.Fatalf("ReplayFiltered nil pred: %v", err)
	}

	if n != 4 {
		t.Errorf("nil pred should replay all 4 events, got %d", n)
	}
}

func TestReplayFiltered_WriteError(t *testing.T) {
	t.Parallel()

	stream := newTestFailingStream(t)

	_, err := sse.ReplayFiltered(stream, testStore(), sse.NewEventID(""),
		func(evt sse.Event) bool { return true })
	if err == nil {
		t.Fatal("expected error on write failure")
	}
}

func TestReplayFiltered_StoreError(t *testing.T) {
	t.Parallel()

	stream, _ := newTestStream(t)

	n, err := sse.ReplayFiltered(stream, failingStore{}, sse.NewEventID(""),
		func(evt sse.Event) bool { return true })
	if err == nil {
		t.Fatal("expected error from failing store")
	}

	if n != 0 {
		t.Errorf("expected 0 on store error, got %d", n)
	}

	if !strings.Contains(err.Error(), "store unavailable") {
		t.Errorf("error should wrap store failure: %v", err)
	}
}

func TestReplayFiltered_AfterGivenID(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	n, err := sse.ReplayFiltered(stream, testStore(), sse.NewEventID("2"),
		func(evt sse.Event) bool { return evt.Event == "message" })
	if err != nil {
		t.Fatalf("ReplayFiltered: %v", err)
	}

	// After ID "2": events 3 (message) and 4 (reaction). Filter to message only → 1.
	if n != 1 {
		t.Errorf("expected 1 message after ID 2, got %d", n)
	}

	body := w.Body.String()

	if !strings.Contains(body, "data: msg2") {
		t.Errorf("missing msg2 in %q", body)
	}

	if strings.Contains(body, "data: react2") {
		t.Errorf("should not contain react2 in %q", body)
	}
}
