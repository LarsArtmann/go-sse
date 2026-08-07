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
	// Returns an error if the underlying store is unavailable (e.g., database failure).
	EventsAfter(lastID EventID) ([]Event, error)
}

// FilteredEventStore is implemented by event stores that can push a predicate
// into their retrieval query, returning only matching events within the replay
// budget. This is the efficient path for [ReplayFiltered]: the replay limit
// applies to MATCHING events, not all events.
//
// Stores that cannot push predicates down (e.g., a simple in-memory ring buffer)
// do not implement this interface; [ReplayFiltered] falls back to
// [EventStore.EventsAfter] + in-memory post-filter.
type FilteredEventStore interface {
	EventStore

	// EventsAfterFiltered returns events with IDs strictly after the given
	// lastID that also satisfy pred, ordered ascending by event ID.
	// Returns an empty slice if no events match or lastID is unknown.
	EventsAfterFiltered(lastID EventID, pred func(Event) bool) ([]Event, error)
}

// Replay sends all events from the store after the given lastEventID
// through the stream. This is used for SSE reconnection: when a client
// reconnects with a Last-Event-ID header, replay the events it missed.
//
// Returns the number of events replayed, or an error if the store fails or
// writing fails.
func Replay(stream *Stream, store EventStore, lastID EventID) (int, error) {
	events, err := store.EventsAfter(lastID)
	if err != nil {
		return 0, errorfamily.Wrapf(err, errorfamily.Rejection,
			"sse.replay_store_failed",
			"retrieve events after %q from store", lastID.Get())
	}

	for i, evt := range events {
		err := stream.Send(evt)
		if err != nil {
			return i, errorfamily.Wrapf(err, errorfamily.Transient,
				"sse.replay_failed",
				"replay after %q: send event %q failed (sent %d of %d)", lastID.Get(), evt.Event, i, len(events))
		}
	}

	return len(events), nil
}

// ReplayFiltered replays only events matching pred from the store. If the store
// implements [FilteredEventStore], the predicate is pushed into the store query
// (efficient: the replay budget is spent entirely on matching events). Otherwise,
// it falls back to [EventStore.EventsAfter] + in-memory post-filter (correct but
// the replay budget may be partially wasted on non-matching events).
//
// In the fallback path, if pred panics, the panic is recovered and treated as a
// non-match (the event is skipped). This mirrors [Broadcaster.SubscribeFilter]'s
// panic-recovery contract.
//
// When pred is nil, ReplayFiltered delegates to [Replay] (no filtering).
//
// Returns the number of events replayed, or an error if the store fails or
// writing fails.
func ReplayFiltered(
	stream *Stream,
	store EventStore,
	lastID EventID,
	pred func(Event) bool,
) (int, error) {
	if pred == nil {
		return Replay(stream, store, lastID)
	}

	var events []Event

	if filtered, ok := store.(FilteredEventStore); ok {
		var err error

		events, err = filtered.EventsAfterFiltered(lastID, pred)
		if err != nil {
			return 0, errorfamily.Wrapf(err, errorfamily.Rejection,
				"sse.replay_store_failed",
				"retrieve filtered events after %q from store (%T)", lastID.Get(), filtered)
		}
	} else {
		all, err := store.EventsAfter(lastID)
		if err != nil {
			return 0, errorfamily.Wrapf(err, errorfamily.Rejection,
				"sse.replay_store_failed",
				"retrieve events after %q from store (%T)", lastID.Get(), store)
		}

		events = make([]Event, 0, len(all))
		for _, evt := range all {
			if safePredCall(pred, evt) {
				events = append(events, evt)
			}
		}
	}

	for i, evt := range events {
		err := stream.Send(evt)
		if err != nil {
			return i, errorfamily.Wrapf(
				err,
				errorfamily.Transient,
				"sse.replay_failed",
				"replay filtered after %q: send event %q failed (sent %d of %d)",
				lastID.Get(),
				evt.Event,
				i,
				len(events),
			)
		}
	}

	return len(events), nil
}
