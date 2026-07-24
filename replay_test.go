package sse_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse"
)

type memoryStore struct {
	events []sse.Event
}

func (m *memoryStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
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

func TestReplay_AfterGivenID(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	store := &memoryStore{
		events: []sse.Event{
			{Event: "item", Data: "first", ID: sse.NewEventID("1")},
			{Event: "item", Data: "second", ID: sse.NewEventID("2")},
			{Event: "item", Data: "third", ID: sse.NewEventID("3")},
			{Event: "item", Data: "fourth", ID: sse.NewEventID("4")},
		},
	}

	n, err := sse.Replay(stream, store, sse.NewEventID("2"))
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	if n != 2 {
		t.Errorf("expected 2 replayed, got %d", n)
	}

	body := w.Body.String()
	if !strings.Contains(body, "id: 3") {
		t.Errorf("missing id: 3 in %q", body)
	}

	if !strings.Contains(body, "data: third") {
		t.Errorf("missing data: third in %q", body)
	}

	if strings.Contains(body, "data: first") {
		t.Errorf("should not contain data: first")
	}

	if strings.Contains(body, "data: second") {
		t.Errorf("should not contain data: second")
	}
}

func TestReplay_EmptyAfterLastEvent(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	store := &memoryStore{
		events: []sse.Event{
			{Event: "item", Data: "first", ID: sse.NewEventID("1")},
		},
	}

	n, err := sse.Replay(stream, store, sse.NewEventID("1"))
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	if n != 0 {
		t.Errorf("expected 0 replayed, got %d", n)
	}
}

func TestReplay_NoLastID(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	store := &memoryStore{
		events: []sse.Event{
			{Event: "item", Data: "first", ID: sse.NewEventID("1")},
			{Event: "item", Data: "second", ID: sse.NewEventID("2")},
		},
	}

	n, err := sse.Replay(stream, store, sse.NewEventID(""))
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	if n != 2 {
		t.Errorf("expected 2, got %d", n)
	}
}

func TestReplay_WriteError(t *testing.T) {
	t.Parallel()

	w := &errorResponseWriter{ResponseWriter: httptest.NewRecorder(), writer: &errorWriter{}}
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	store := &memoryStore{
		events: []sse.Event{
			{Event: "item", Data: "first", ID: sse.NewEventID("1")},
		},
	}

	_, err := sse.Replay(stream, store, sse.NewEventID(""))
	if err == nil {
		t.Fatal("expected error on write failure")
	}
}

// errorResponseWriter wraps errorWriter as http.ResponseWriter.
type errorResponseWriter struct {
	http.ResponseWriter
	writer *errorWriter
}

func (e *errorResponseWriter) Write(p []byte) (int, error) {
	return e.writer.Write(p)
}

func (e *errorResponseWriter) Flush() {}

// failingStore always returns an error from EventsAfter.
type failingStore struct{}

func (failingStore) EventsAfter(sse.EventID) ([]sse.Event, error) {
	return nil, errors.New("store unavailable")
}

func TestReplay_StoreError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	n, err := sse.Replay(stream, failingStore{}, sse.NewEventID(""))
	if err == nil {
		t.Fatal("expected error from failing store")
	}

	if n != 0 {
		t.Errorf("expected 0 replayed on store error, got %d", n)
	}

	if !strings.Contains(err.Error(), "store unavailable") {
		t.Errorf("error should wrap store failure: %v", err)
	}
}
