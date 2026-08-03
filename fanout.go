package sse

import (
	"context"
	"reflect"
	"sync"
)

// defaultSubscriberBuffer is the per-subscriber channel capacity. Broadcasts
// are non-blocking: a subscriber whose buffer is full has events dropped.
// 64 is large enough to absorb short bursts without dropping under normal
// fan-out, while bounding memory per subscriber.
const defaultSubscriberBuffer = 64

// Option configures a Broadcaster at construction time. Use the helpers
// ([WithBufferSize]) rather than constructing an Option literal directly.
type Option[T any] func(*fanOut[T])

// WithBufferSize overrides the per-subscriber channel capacity. The default
// is [defaultSubscriberBuffer] (64). Pass any positive integer; values ≤ 0
// are ignored and the default is kept.
//
// Larger buffers absorb longer consumer slow-downs before drops begin; smaller
// buffers reduce memory per subscriber at the cost of earlier drops. The
// setting applies to subscribers created after NewBroadcaster returns; it is
// read once and not changed later.
//
//	func NewBroadcaster[T any](opts ...Option[T]) signature uses generics.
func WithBufferSize[T any](size int) Option[T] {
	return func(f *fanOut[T]) {
		if size > 0 {
			f.bufferSize = size
		}
	}
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
	onSubscribe   func()
	onUnsubscribe func()
}

func newFanOut[T any](opts ...Option[T]) *fanOut[T] {
	f := &fanOut[T]{
		mu:            sync.RWMutex{},
		subscribers:   make(map[uintptr]*subscriber[T]),
		bufferSize:    defaultSubscriberBuffer,
		onSubscribe:   nil,
		onUnsubscribe: nil,
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
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
// After [Broadcaster.Close], Subscribe returns a closed channel (no-op).
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
// After [Broadcaster.Close], SubscribeFilter returns a closed channel (no-op).
func (f *fanOut[T]) SubscribeFilter(pred func(T) bool) <-chan T {
	ch := make(chan T, f.effectiveBufferSize())

	f.mu.Lock()
	if f.subscribers == nil {
		f.mu.Unlock()
		close(ch) // already closed — return a closed channel

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

// sendAllLocked fans msg out to every subscriber. The caller must hold f.mu
// (read or write) so that a concurrent Unsubscribe cannot close a channel mid-send.
func (f *fanOut[T]) sendAllLocked(msg T) {
	for _, sub := range f.subscribers {
		if sub.pred != nil && !sub.pred(msg) {
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

	for key, sub := range f.subscribers {
		delete(f.subscribers, key)
		close(sub.ch)
	}

	f.subscribers = nil // marks as closed
}

// drainLocked waits for every active subscriber's channel to be empty (the
// consumer caught up and read everything) or for ctx to be cancelled. The
// caller must hold f.mu; this function releases it before blocking, then
// re-acquires it to close the subscribers and mark the hub closed.
//
// On success, all subscriber channels are closed and f.subscribers is set
// to nil (the closed sentinel). On context cancellation, the function returns
// ctx.Err() without closing anything; subsequent Subscribe calls still
// return live channels.
func (f *fanOut[T]) drainLocked(ctx context.Context) error {
	if f.subscribers == nil {
		return nil // already closed — nothing to drain
	}

	// Snapshot the subscriber set under the lock; we need to wait outside it.
	subs := make([]*subscriber[T], 0, len(f.subscribers))
	for _, sub := range f.subscribers {
		subs = append(subs, sub)
	}

	f.mu.Unlock()
	defer f.mu.Lock()

	for {
		// If a subscriber disconnected while we were waiting, len(f.subscribers)
		// shrank but our snapshot still holds a reference to the (now-closed)
		// channel. Drain check below tolerates closed channels: a receive on a
		// closed, empty channel returns the zero value with ok=false, which
		// our len() check would not see — but we also re-acquire the lock to
		// verify the live subscriber set. See Shutdown test.
		allDrained := true

		for _, sub := range subs {
			// Use len() to avoid a blocking receive. Channels report their
			// current buffered length; a subscriber with len == 0 has
			// consumed everything that was sent.
			if len(sub.ch) > 0 {
				allDrained = false

				break
			}
		}

		if allDrained {
			// Re-acquire the lock (via the defer) and close everything.
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-afterBriefDelay():
			// Re-check drain status. Polling is required because the
			// consumer's reads are not observable from here.
		}
	}

	// Lock is held by the deferred f.mu.Lock() above. Close the snapshot.
	for _, sub := range subs {
		// The channel might have been closed by Unsubscribe in the meantime;
		// that's fine — closing an already-closed channel would panic, so
		// check first.
		select {
		case _, ok := <-sub.ch:
			if !ok {
				continue // already closed by Unsubscribe
			}
		default:
		}

		close(sub.ch)
	}

	f.subscribers = nil

	return nil
}

// afterBriefDelay returns a channel that fires after a short poll interval.
// The poll is intentionally short so drain latency is bounded but not so
// short that it dominates CPU when there are many slow consumers.
func afterBriefDelay() <-chan struct{} {
	ch := make(chan struct{})

	go func() {
		defer close(ch)
		// 1ms is a balance: long enough to avoid burning CPU, short enough
		// that consumer reads register promptly.
		t := time.NewTimer(time.Millisecond)
		defer t.Stop()
		<-t.C
	}()

	return ch
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
