package sse

import (
	"context"
	"maps"
	"slices"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Shutdown gracefully drains the broadcaster: it stops accepting new
// subscribers, waits for every active subscriber's buffer to be empty
// (consumers caught up), then closes all subscriber channels and marks
// the hub as closed. Returns nil on a clean drain.
//
// If ctx is cancelled before the drain completes, Shutdown returns
// ctx.Err() without closing anything. The broadcaster remains in
// "draining" state; the caller can either retry with a fresh context
// or call [Close] to abandon the drain and shut down immediately.
//
// While Shutdown is in progress, Subscribe returns a closed channel
// (no new work piles up). Existing subscribers continue to receive
// broadcasts until the drain completes or the context is cancelled.
//
// Shutdown is safe to call from multiple goroutines; only the first
// call's drain takes effect, and subsequent calls return nil once
// the hub is already closed.
func (f *fanOut[T]) Shutdown(ctx context.Context) error {
	f.mu.Lock()
	if f.subscribers == nil {
		// Already closed (via Close or a prior successful Shutdown).
		f.mu.Unlock()

		return nil
	}

	// Snapshot the subscriber set; we need to wait outside the lock.
	subs := slices.Collect(maps.Values(f.subscribers))

	// Mark draining so new Subscribe calls return a closed channel.
	f.draining = true

	f.mu.Unlock()

	// Wait for each snapshot subscriber's buffer to empty, or ctx to fire.
	if err := f.waitForDrain(ctx, subs); err != nil {
		return errorfamily.Wrapf(err, errorfamily.Transient,
			"sse.shutdown_failed",
			"drain %d subscribers", len(subs))
	}

	// Drain complete. Re-acquire the lock to close any subscribers that
	// haven't already been closed by Unsubscribe. A subscriber that
	// disconnected during the drain is no longer in f.subscribers (it
	// was removed by Unsubscribe), so closing the snapshot would close
	// an already-closed channel. We tolerate that by checking the map.
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, sub := range subs {
		if _, stillTracked := f.subscribers[channelPtr(sub.ch)]; stillTracked {
			delete(f.subscribers, channelPtr(sub.ch))
			close(sub.ch)
		}
	}

	f.subscribers = nil
	f.draining = false

	return nil
}

// waitForDrain blocks until every subscriber's buffer is empty or ctx is
// cancelled. Returns nil once all buffers drain, or a wrapped context error
// if the deadline fires first. Uses index-based loops and time.After (not a
// named ticker) so no structural variables leak into the error scope.
func (f *fanOut[T]) waitForDrain(ctx context.Context, subs []*subscriber[T]) error {
	for {
		notDrained := 0

		for i := range subs {
			if len(subs[i].ch) > 0 {
				notDrained++
			}
		}

		if notDrained == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			// Drain deadline exceeded. Leave draining=true so Subscribe
			// continues to reject new subscribers; the caller can either
			// retry Shutdown with a fresh context or call Close.
			return errorfamily.Wrapf(
				ctx.Err(),
				errorfamily.Transient,
				"sse.shutdown_drain_deadline_exceeded",
				"broadcaster drain did not complete before context deadline: %d of %d subscribers still have buffered events",
				notDrained,
				len(subs),
			)
		case <-time.After(drainPollInterval):
		}
	}
}

// Health returns a snapshot of the broadcaster's lifecycle state. It is
// safe to call concurrently with any other method. The returned struct
// is a value type so it cannot accidentally mutate the broadcaster.
func (f *fanOut[T]) Health() BroadcasterHealth {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return BroadcasterHealth{
		Closed:          f.subscribers == nil,
		Draining:        f.draining,
		SubscriberCount: len(f.subscribers),
		BufferSize:      f.effectiveBufferSize(),
	}
}

// OnSubscribe registers a callback fired after each successful Subscribe.
// Used for connection metrics, logging, or triggering initial state sends.
// Pass nil to clear a previously registered callback.
func (f *fanOut[T]) OnSubscribe(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.onSubscribe = fn
}

// OnUnsubscribe registers a callback fired after each successful Unsubscribe.
// Used for disconnection metrics or cleanup logging.
// Pass nil to clear a previously registered callback.
func (f *fanOut[T]) OnUnsubscribe(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.onUnsubscribe = fn
}

// OnDrop registers a callback fired each time a message is dropped because a
// subscriber's buffer is full. The callback receives the dropped message and
// runs on the broadcasting goroutine inside the fan-out read lock, so it must
// be fast and non-blocking. Pass nil to clear a previously registered callback.
//
// The callback must not re-enter the broadcaster: it runs under the read lock,
// while this setter takes the write lock — a re-entrant call deadlocks. A
// panicking callback is recovered and the broadcast continues; see [WithOnDrop]
// for the full contract.
//
// The callback fires once per full subscriber per broadcast: a single Broadcast
// to N full subscribers invokes onDrop N times with the same msg.
func (f *fanOut[T]) OnDrop(fn func(T)) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.onDrop = fn
}
