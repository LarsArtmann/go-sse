package main

import (
	"strconv"
	"testing"

	"github.com/larsartmann/go-sse"
)

func TestMemStore_EmptyStore(t *testing.T) {
	t.Parallel()

	store := newMemStore(maxStoredEvents)

	events, err := store.EventsAfter(sse.NewEventID("0"))
	if err != nil {
		t.Fatalf("EventsAfter on empty store: unexpected error: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("EventsAfter on empty store: expected 0 events, got %d", len(events))
	}
}

func TestMemStore_InvalidID(t *testing.T) {
	t.Parallel()

	store := newMemStore(maxStoredEvents)
	store.Append(makeTestEvent(1))
	store.Append(makeTestEvent(2))

	tests := []struct {
		name  string
		idStr string
		want  int
	}{
		{"non-integer", "abc", 2},
		{"empty string", "", 2},
		{"very large ID", "999999", 0},
		{"zero ID gets all", "0", 2},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			events, err := store.EventsAfter(sse.NewEventID(testCase.idStr))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(events) != testCase.want {
				t.Errorf(
					"id=%q: expected %d events, got %d",
					testCase.idStr,
					testCase.want,
					len(events),
				)
			}
		})
	}
}

func TestMemStore_SequentialReplay(t *testing.T) {
	t.Parallel()

	store := newMemStore(maxStoredEvents)

	for i := range 5 {
		store.Append(makeTestEvent(int64(i + 1)))
	}

	events, err := store.EventsAfter(sse.NewEventID("2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events after ID 2, got %d", len(events))
	}

	for i, evt := range events {
		expectedID := strconv.FormatInt(int64(i+3), 10)
		if evt.ID.Get() != expectedID {
			t.Errorf("event %d: expected ID %s, got %s", i, expectedID, evt.ID.Get())
		}
	}
}

func TestMemStore_RingBufferEviction(t *testing.T) {
	t.Parallel()

	capacity := 5
	store := newMemStore(capacity)

	total := capacity + 10
	for i := range total {
		store.Append(makeTestEvent(int64(i + 1)))
	}

	events, err := store.EventsAfter(sse.NewEventID("0"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != capacity {
		t.Fatalf("expected %d events after eviction, got %d", capacity, len(events))
	}

	firstID := events[0].ID.Get()
	expectedFirst := "11"
	if firstID != expectedFirst {
		t.Errorf("expected first event ID %s after eviction, got %s", expectedFirst, firstID)
	}

	lastID := events[len(events)-1].ID.Get()
	expectedLast := "15"
	if lastID != expectedLast {
		t.Errorf("expected last event ID %s after eviction, got %s", expectedLast, lastID)
	}
}
