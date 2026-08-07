package sse

import (
	"context"
	"reflect"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// defaultSubscriberBuffer is the per-subscriber channel capacity. Broadcasts
// are non-blocking: a subscriber whose buffer is full has events dropped.
// 64 is large enough to absorb short bursts without dropping under normal
// fan-out, while bounding memory per subscriber.
const defaultSubscriberBuffer = 64

// drainPollInterval is how often [fanOut.shutdownLocked] re-checks whether
// all subscriber buffers have been drained while waiting for consumers to
// catch up. Short enough that an idle consumer registers a drain promptly,
// long enough to avoid burning CPU.
const drainPollInterval = time.Millisecond

// Option configures a Broadcaster at construction time. Use the helpers
// ([WithBufferSize]) rather than constructing an Option literal directly.
type Option[T any] func(*fanOut[T])

// WithBufferSize overrides the per-subscriber channel capacity. The default
// is [defaultSubscriberBuffer] (64). Pass any positive integer; values ≤ 0
// are ignored and the default is kept.
//
// Larger buffers absorb longer consumer slow-downs before drops begin; smaller
// buffers reduce memory per subscriber at the cost of earlier drops. The
// setting is read at construction time and not changed later.
func WithBufferSize[T any](size int) Option[T] {
	return func(f *fanOut[T]) {
		if size > 0 {
			f.bufferSize = size
		}
	}
}

// BroadcasterHealth is a snapshot of a [Broadcaster]'s lifecycle state,
// returned by [Broadcaster.Health]. It is the structured-status counterpart
// to the unstructured boolean [Broadcaster.SubscriberCount]; consumers wire
// it into health checks (k8s liveness/readiness, load balancer probes,
// on-call dashboards) without depending on the unexported internals.
type BroadcasterHealth struct {
	// Closed is true after [Broadcaster.Close] or a successful
	// [Broadcaster.Shutdown]. A closed broadcaster rejects new
	// subscribers (Subscribe returns a closed channel) and silently
	// drops broadcasts.
	Closed bool

	// Draining is true while [Broadcaster.Shutdown] is waiting for
	// subscriber buffers to drain. During draining, new Subscribe
	// calls return a closed channel so no new work piles up; existing
	// subscribers are still alive and will be closed once the drain
	// completes (or the context is cancelled).
	Draining bool

	// SubscriberCount is the number of currently registered subscribers.
	// Does not count subscribers that disconnected during a drain.
	SubscriberCount int

	// BufferSize is the per-subscriber channel capacity, in events.
	// Defaults to [defaultSubscriberBuffer] (64). Configurable via
	// [WithBufferSize] at construction time.
	BufferSize int
}

// subscriber wraps a subscriber channel with an optional predicate.
// When pred is nil, all events are delivered (the default unfiltered case).
// When pred is non-nil, only events for which pred(msg) returns true are sent.
type subscriber[T any] struct {
	ch   chan T
	pred func(T) bool // nil = all events
}

// fanOut is the transport-agnostic subscriber hub shared by [Broadcaster].
// It provides thread-safe fan-out with O(1) unsubscribe via channel pointer
// identity and non-blocking broadcast (drops to slow consumers).
type fanOut[T any] struct {
	mu            sync.RWMutex
	subscribers   map[uintptr]*subscriber[T]
	bufferSize    int
	draining      bool // true while Shutdown is in progress; rejects new Subscribe
	onSubscribe   func()
	onUnsubscribe func()
}

func newFanOut[T any](opts ...Option[T]) *fanOut[T] {
	hub := &fanOut[T]{
		mu:            sync.RWMutex{},
		subscribers:   make(map[uintptr]*subscriber[T]),
		bufferSize:    defaultSubscriberBuffer,
		draining:      false,
		onSubscribe:   nil,
		onUnsubscribe: nil,
	}

	for _, opt := range opts {
		opt(hub)
	}

	return hub
}

// effectiveBufferSize returns the configured buffer size, falling back to
// the default if none was set or the configured value is non-positive.
func (f *fanOut[T]) effectiveBufferSize() int {
	if f.bufferSize > 0 {
		return f.bufferSize
	}

	return defaultSubscriberBuffer
}

// Subscribe creates a new subscriber channel that receives all broadcast
// messages. The channel has a buffer of 64 by default (configurable via
// [WithBufferSize]); slower consumers may miss messages when the buffer is
// full.
//
// Call [Broadcaster.Unsubscribe] when the client disconnects to prevent memory leaks.
// After [Broadcaster.Close] or during [Broadcaster.Shutdown], Subscribe returns
// a closed channel (no-op).
func (f *fanOut[T]) Subscribe() <-chan T {
	return f.SubscribeFilter(nil)
}

// SubscribeFilter creates a new subscriber channel that receives only broadcast
// messages for which pred returns true. When pred is nil, all events are
// delivered (identical to [Subscribe]).
//
// The predicate is called inside the fan-out loop under the read lock — it must
// be pure, fast, and non-blocking. It is called once per subscriber per
// broadcast, not once per event globally.
//
// Call [Broadcaster.Unsubscribe] when the client disconnects to prevent memory leaks.
// After [Broadcaster.Close] or during [Broadcaster.Shutdown], SubscribeFilter
// returns a closed channel (no-op).
func (f *fanOut[T]) SubscribeFilter(pred func(T) bool) <-chan T {
	ch := make(chan T, f.effectiveBufferSize())

	f.mu.Lock()
	if f.subscribers == nil || f.draining {
		f.mu.Unlock()
		close(ch) // already closed or shutting down — return a closed channel

		return ch
	}

	f.subscribers[channelPtr(ch)] = &subscriber[T]{ch: ch, pred: pred}
	onSub := f.onSubscribe
	f.mu.Unlock()

	if onSub != nil {
		onSub()
	}

	return ch
}

// Unsubscribe removes a subscriber channel and closes it.
// Call this when a client disconnects to prevent memory leaks.
func (f *fanOut[T]) Unsubscribe(ch <-chan T) {
	f.mu.Lock()
	key := channelPtr(ch)

	sub, ok := f.subscribers[key]
	if ok {
		delete(f.subscribers, key)
		close(sub.ch)
	}

	onUnsub := f.onUnsubscribe
	f.mu.Unlock()

	if ok && onUnsub != nil {
		onUnsub()
	}
}

// Broadcast sends a message to all active subscribers.
// Slow subscribers with full buffers have the message dropped to prevent
// blocking the broadcaster.
//
// The iteration runs under the read lock so that a concurrent Unsubscribe
// cannot close a channel that this loop is about to send on — sending to a
// closed channel would panic. Because sends use a non-blocking select, no
// goroutine blocks here, and the lock is held only for the brief fan-out.
//
// After [Broadcaster.Close] the subscriber set is nil and broadcasts are
// silently dropped. During [Broadcaster.Shutdown] the set is still live,
// so broadcasts reach the existing subscribers until the drain completes.
func (f *fanOut[T]) Broadcast(msg T) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	f.sendAllLocked(msg)
}

// BroadcastMany sends multiple messages to all active subscribers in a single
// locked fan-out pass. Per-subscriber ordering is preserved across the batch,
// and the read lock is acquired once instead of once per message — this is
// meaningfully cheaper than calling Broadcast in a loop for large batches.
// Like Broadcast, slow subscribers with full buffers have individual messages
// dropped (non-blocking).
func (f *fanOut[T]) BroadcastMany(msgs ...T) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, msg := range msgs {
		f.sendAllLocked(msg)
	}
}

// safePredCall calls pred(msg), recovering from panics. A panicking predicate
// returns false (treated as "no match") so one broken predicate cannot crash
// the broadcaster or replay loop. The zero overhead for unfiltered subscribers
// (nil pred) is preserved — callers check nil before calling this helper.
//
//nolint:nonamedreturns // named return required so the deferred recover can set the result
func safePredCall[T any](pred func(T) bool, msg T) (match bool) {
	defer func() {
		if r := recover(); r != nil {
			match = false
		}
	}()

	return pred(msg)
}

// sendAllLocked fans msg out to every subscriber. The caller must hold f.mu
// (read or write) so that a concurrent Unsubscribe cannot close a channel mid-send.
func (f *fanOut[T]) sendAllLocked(msg T) {
	for _, sub := range f.subscribers {
		if sub.pred != nil && !safePredCall(sub.pred, msg) {
			continue
		}

		select {
		case sub.ch <- msg:
		default:
		}
	}
}

// SubscriberCount returns the number of active subscribers.
func (f *fanOut[T]) SubscriberCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return len(f.subscribers)
}

// Close shuts down the fan-out hub: it closes all subscriber channels and
// marks the hub as closed so that future Subscribe calls return a closed
// channel (no-op). Broadcasts after Close are silently dropped.
//
// Close is the instant shutdown primitive — use it when you do not need to
// wait for in-flight events to be consumed. For graceful shutdown that drains
// subscriber buffers first, use [Broadcaster.Shutdown] with a context.
func (f *fanOut[T]) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.closeLocked()
}

// closeLocked closes every active subscriber and marks the hub closed.
// The caller must hold f.mu (write).
func (f *fanOut[T]) closeLocked() {
	for key, sub := range f.subscribers {
		delete(f.subscribers, key)
		close(sub.ch)
	}

	f.subscribers = nil
}

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
	subs := make([]*subscriber[T], 0, len(f.subscribers))
	for _, sub := range f.subscribers {
		subs = append(subs, sub)
	}

	// Mark draining so new Subscribe calls return a closed channel.
	f.draining = true

	f.mu.Unlock()

	// Wait for each snapshot subscriber's buffer to empty, or ctx to fire.
	ticker := time.NewTicker(drainPollInterval)
	defer ticker.Stop()

	for {
		allDrained := true

		for _, sub := range subs {
			if len(sub.ch) > 0 {
				allDrained = false

				break
			}
		}

		if allDrained {
			break
		}

		select {
		case <-ctx.Done():
			// Drain deadline exceeded. Leave draining=true so Subscribe
			// continues to reject new subscribers; the caller can either
			// retry Shutdown with a fresh context or call Close.
			notDrained := 0

			for _, sub := range subs {
				if len(sub.ch) > 0 {
					notDrained++
				}
			}

			return errorfamily.Wrapf(ctx.Err(), errorfamily.Transient,
				"sse.shutdown_drain_deadline_exceeded",
				"broadcaster drain did not complete before context deadline: %d of %d subscribers still have buffered events",
				notDrained, len(subs))
		case <-ticker.C:
		}
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

// channelPtr returns the pointer identity of a channel, regardless of direction.
func channelPtr(ch any) uintptr {
	return reflect.ValueOf(ch).Pointer()
}
