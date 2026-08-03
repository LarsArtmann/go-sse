package sse

import (
	"context"
)

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
// # Filtered Subscriptions
//
// [Broadcaster.SubscribeFilter] subscribes with a predicate: only messages for
// which the predicate returns true are delivered to the subscriber's channel.
// The predicate is checked before the non-blocking send, so irrelevant messages
// never enter the subscriber's buffer. This is useful for per-channel,
// per-tenant, or per-event-type routing without multiple broadcasters.
//
//	// Deliver only "message" events
//	ch := broadcaster.SubscribeFilter(func(evt sse.Event) bool {
//	    return evt.Event == "message"
//	})
//	defer broadcaster.Unsubscribe(ch)
//
// The predicate is called inside the fan-out loop under the read lock — it must
// be pure, fast, and non-blocking. If a predicate panics, the panic is recovered
// and treated as a non-match (the message is skipped for that subscriber). This
// ensures one broken predicate cannot crash the broadcaster. Subscribe() with
// no filter is equivalent to SubscribeFilter(nil).
//
// # Backpressure and Drop Policy
//
// Broadcast is non-blocking. If a subscriber's channel buffer is full (default
// capacity 64), the message is silently dropped for that subscriber. This
// prevents one slow consumer from blocking the entire fan-out. Consumers that
// need guaranteed delivery should implement application-level ack/retry.
//
// The buffer size is configurable via [WithBufferSize] for workloads that
// tolerate more buffering (fewer drops, more memory) or less (more drops, less
// memory per subscriber).
//
// # Lifecycle
//
// Broadcaster exposes two shutdown paths:
//
//   - [Broadcaster.Close] — instant: closes every subscriber channel and
//     marks the hub closed. Use during hard shutdown when in-flight events
//     do not need to be delivered.
//   - [Broadcaster.Shutdown] — graceful: stops accepting new subscribers,
//     waits for every active subscriber's buffer to drain (consumers catch
//     up), then closes the channels. Returns ctx.Err() if the context fires
//     before the drain completes; the caller can retry with a fresh context
//     or fall back to Close.
//
// [Broadcaster.Health] returns a structured snapshot for health checks
// (k8s liveness/readiness, load balancer probes).
type Broadcaster[T any] struct {
	*fanOut[T]
}

// NewBroadcaster creates a new broadcaster with no subscribers. Pass any
// number of [Option] values to configure the broadcaster; see [WithBufferSize]
// for the most common one.
//
//	func NewBroadcaster[sse.Event](sse.WithBufferSize[sse.Event](256))
func NewBroadcaster[T any](opts ...Option[T]) *Broadcaster[T] {
	return &Broadcaster[T]{fanOut: newFanOut[T](opts...)}
}

// BroadcastMany sends a batch of messages to all active subscribers in a single
// locked fan-out pass. It is the batch variant of [Broadcaster.Broadcast]:
// cheaper than looping Broadcast (one lock acquisition) and it preserves
// per-subscriber ordering across the batch. Slow subscribers with full buffers
// have individual messages dropped, exactly like Broadcast.
func (b *Broadcaster[T]) BroadcastMany(msgs ...T) {
	b.fanOut.BroadcastMany(msgs...)
}

// Shutdown gracefully drains the broadcaster. See [fanOut.Shutdown] for
// the full contract; this is a thin pass-through that surfaces the method
// on the public Broadcaster type.
//
// Example usage with a SIGTERM handler:
//
//	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
//	defer cancel()
//	if err := broadcaster.Shutdown(ctx); err != nil {
//	    // Drain deadline exceeded; fall back to instant close.
//	    broadcaster.Close()
//	}
func (b *Broadcaster[T]) Shutdown(ctx context.Context) error {
	return b.fanOut.Shutdown(ctx)
}

// Health returns a snapshot of the broadcaster's lifecycle state. Safe to
// call from any goroutine and from a health check loop.
func (b *Broadcaster[T]) Health() BroadcasterHealth {
	return b.fanOut.Health()
}
