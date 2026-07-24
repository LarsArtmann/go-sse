package sse

// Broadcaster distributes messages to all subscribed clients via Go channels.
// It is safe for concurrent use and generic over the message type.
//
// Create one at application startup and share it across handlers:
//
//	broadcaster := sse.NewBroadcaster[sse.Event]()
//
//	// SSE endpoint handler:
//	ch := broadcaster.Subscribe()
//	defer broadcaster.Unsubscribe(ch)
//
//	// Push from anywhere:
//	broadcaster.Broadcast(sse.Event{Event: "update", Data: "<div>new</div>"})
//
// # Backpressure and Drop Policy
//
// Broadcast is non-blocking. If a subscriber's channel buffer is full (default
// capacity 64), the message is silently dropped for that subscriber. This
// prevents one slow consumer from blocking the entire fan-out. Consumers that
// need guaranteed delivery should implement application-level ack/retry.
type Broadcaster[T any] struct {
	*fanOut[T]
}

// NewBroadcaster creates a new broadcaster with no subscribers.
func NewBroadcaster[T any]() *Broadcaster[T] {
	return &Broadcaster[T]{fanOut: newFanOut[T]()}
}

// BroadcastMany sends a batch of messages to all active subscribers in a single
// locked fan-out pass. It is the batch variant of [Broadcaster.Broadcast]:
// cheaper than looping Broadcast (one lock acquisition) and it preserves
// per-subscriber ordering across the batch. Slow subscribers with full buffers
// have individual messages dropped, exactly like Broadcast.
func (b *Broadcaster[T]) BroadcastMany(msgs ...T) {
	b.fanOut.BroadcastMany(msgs...)
}
