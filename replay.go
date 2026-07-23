package sse

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// EventStore retrieves events for SSE reconnection replay.
// Implementations must be safe for concurrent access.
type EventStore interface {
	// EventsAfter returns events with IDs strictly after the given lastID.
	// Returns an empty slice if no events are found or lastID is unknown.
	// The returned slice must be ordered by event ID (ascending).
	EventsAfter(lastID EventID) []Event
}

// Replay sends all events from the store after the given lastEventID
// through the stream. This is used for SSE reconnection: when a client
// reconnects with a Last-Event-ID header, replay the events it missed.
//
// Returns the number of events replayed, or an error if writing fails.
func Replay(stream *Stream, store EventStore, lastID EventID) (int, error) {
	events := store.EventsAfter(lastID)

	for i, evt := range events {
		err := stream.Send(evt)
		if err != nil {
			return i, errorfamily.Wrapf(err, errorfamily.Transient,
				"sse.replay_failed",
				"replay after %q (sent %d of %d)", lastID.Get(), i, len(events))
		}
	}

	return len(events), nil
}
