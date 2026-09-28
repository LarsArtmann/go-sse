package ssetest

import "slices"

// FindByType returns the first event whose type matches, along with true.
// Returns false if no match is found. An empty type matches unnamed events
// (the browser "message" default).
//
// Useful when a handler sends multiple event types and you need to assert on a
// specific one without indexing by position.
func FindByType(events []Event, eventType string) (Event, bool) {
	for _, evt := range events {
		if evt.Type == eventType {
			return evt, true
		}
	}

	return Event{}, false
}

// FilterByType returns only the events whose type matches. An empty type
// selects unnamed events (the browser "message" default).
func FilterByType(events []Event, eventType string) []Event {
	return slices.DeleteFunc(slices.Clone(events), func(e Event) bool {
		return e.Type != eventType
	})
}
