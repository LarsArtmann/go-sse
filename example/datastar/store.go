package main

import (
	"strconv"
	"sync"
	"time"

	"github.com/larsartmann/go-sse"
)

// memStore is an in-memory ring buffer implementing sse.EventStore.
// It keeps the last maxStoredEvents events for reconnection replay.
// Timestamps are tracked in parallel for the ?since= duration filter.
type memStore struct {
	mu          sync.RWMutex
	events      []sse.Event
	timestamps  []time.Time
	cap         int
}

func newMemStore(capacity int) *memStore {
	return &memStore{
		mu:     sync.RWMutex{},
		events: make([]sse.Event, 0, capacity),
		cap:    capacity,
	}
}

func (s *memStore) Append(evt sse.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, evt)
	s.timestamps = append(s.timestamps, time.Now())

	if len(s.events) > s.cap {
		s.events = s.events[len(s.events)-s.cap:]
		s.timestamps = s.timestamps[len(s.timestamps)-s.cap:]
	}
}

func (s *memStore) EventsAfter(lastID sse.EventID) ([]sse.Event, error) {
	// O(n) linear scan — fine for a 50-event demo ring buffer.
	// A production store would index events by ID for O(log n) lookup.
	lastSeq, err := strconv.Atoi(lastID.Get())
	if err != nil {
		lastSeq = 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []sse.Event

	for _, evt := range s.events {
		seq, err := strconv.Atoi(evt.ID.Get())
		if err != nil {
			continue
		}

		if seq > lastSeq {
			result = append(result, evt)
		}
	}

	return result, nil
}

// EventsAfterSince returns events after lastID that also fall within the
// given duration from now. Used by the ?since= query parameter to filter
// replayed events by age (e.g., ?since=5m replays only the last 5 minutes).
func (s *memStore) EventsAfterSince(lastID sse.EventID, since time.Duration) ([]sse.Event, error) {
	lastSeq, err := strconv.Atoi(lastID.Get())
	if err != nil {
		lastSeq = 0
	}

	cutoff := time.Now().Add(-since)

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []sse.Event

	for i, evt := range s.events {
		seq, err := strconv.Atoi(evt.ID.Get())
		if err != nil {
			continue
		}

		if seq > lastSeq && s.timestamps[i].After(cutoff) {
			result = append(result, evt)
		}
	}

	return result, nil
}
