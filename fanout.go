package sse

import (
	"reflect"
	"sync"
)

// defaultSubscriberBuffer is the per-subscriber channel capacity. Broadcasts
// are non-blocking: a subscriber whose buffer is full has events dropped.
// 64 is large enough to absorb short bursts without dropping under normal
// fan-out, while bounding memory per subscriber.
const defaultSubscriberBuffer = 64

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
	onSubscribe   func()
	onUnsubscribe func()
}

func newFanOut[T any]() *fanOut[T] {
	return &fanOut[T]{
		mu:            sync.RWMutex{},
		subscribers:   make(map[uintptr]*subscriber[T]),
		onSubscribe:   nil,
		onUnsubscribe: nil,
	}
}

// Subscribe creates a new subscriber channel that receives all broadcast
// messages. The channel has a buffer of 64; slower consumers may miss messages
// when the buffer is full.
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
	ch := make(chan T, defaultSubscriberBuffer)

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
// This is the graceful-shutdown primitive for broadcasters. Call it
// when your server is shutting down so connected clients receive a channel-close
// signal and their read loops exit cleanly.
func (f *fanOut[T]) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()

	for key, sub := range f.subscribers {
		delete(f.subscribers, key)
		close(sub.ch)
	}

	f.subscribers = nil // marks as closed
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
