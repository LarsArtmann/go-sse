package main

import (
	"strconv"
	"sync"

	"github.com/larsartmann/go-sse"
)

// memStore is an in-memory ring buffer implementing sse.EventStore.
// It keeps the last maxStoredEvents events for reconnection replay.
type memStore struct {
	mu     sync.RWMutex
	events []sse.Event
	cap    int
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

	if len(s.events) > s.cap {
		s.events = s.events[len(s.events)-s.cap:]
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
