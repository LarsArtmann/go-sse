package sse_test

import (
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-sse"
)

func TestWithOnDrop_FiresWhenBufferFull(t *testing.T) {
	t.Parallel()

	var drops atomic.Int64

	bc := sse.NewBroadcaster[int](
		sse.WithBufferSize[int](1), // buffer of 1 — second send drops
		sse.WithOnDrop[int](func(msg int) {
			drops.Add(1)
		}),
	)
	t.Cleanup(func() { bc.Close() })

	// Subscribe but never drain — the buffer fills after 1 message.
	ch := bc.Subscribe()
	t.Cleanup(func() { bc.Unsubscribe(ch) })

	bc.Broadcast(1) // fills the buffer (len 1)
	bc.Broadcast(2) // buffer full — drop fires
	bc.Broadcast(3) // buffer full — drop fires

	if got := drops.Load(); got != 2 {
		t.Errorf("expected 2 drops, got %d", got)
	}
}

func TestWithOnDrop_NoCallbackNoPanic(t *testing.T) {
	t.Parallel()

	bc := sse.NewBroadcaster[int](
		sse.WithBufferSize[int](0), // no buffer
	)
	t.Cleanup(func() { bc.Close() })

	ch := bc.Subscribe()
	t.Cleanup(func() { bc.Unsubscribe(ch) })

	// Should not panic even though no onDrop is set.
	bc.Broadcast(1)
	bc.Broadcast(2)
}

func TestOnDrop_RuntimeRegistration(t *testing.T) {
	t.Parallel()

	var drops atomic.Int64

	bc := sse.NewBroadcaster[int](
		sse.WithBufferSize[int](1),
	)
	t.Cleanup(func() { bc.Close() })

	// Register at runtime, not construction time.
	bc.OnDrop(func(msg int) {
		drops.Add(1)
	})

	ch := bc.Subscribe()
	t.Cleanup(func() { bc.Unsubscribe(ch) })

	bc.Broadcast(1) // fills the buffer
	bc.Broadcast(2) // buffer full — drop fires

	if got := drops.Load(); got != 1 {
		t.Errorf("expected 1 drop, got %d", got)
	}
}
