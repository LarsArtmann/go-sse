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
		sse.WithBufferSize[int](1), // buffer of 1 — second send hits the drop branch
	)
	t.Cleanup(func() { bc.Close() })

	ch := bc.Subscribe()
	t.Cleanup(func() { bc.Unsubscribe(ch) })

	// Should not panic even though no onDrop is set.
	bc.Broadcast(1) // fills the buffer
	bc.Broadcast(2) // buffer full — hits the nil-onDrop drop branch
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

func TestWithOnDrop_FiresPerSubscriber(t *testing.T) {
	t.Parallel()

	var drops atomic.Int64

	bc := sse.NewBroadcaster[int](
		sse.WithBufferSize[int](1),
		sse.WithOnDrop[int](func(msg int) {
			drops.Add(1)
		}),
	)
	t.Cleanup(func() { bc.Close() })

	// Three subscribers, none of which ever drain.
	const subs = 3
	chans := make([]<-chan int, 0, subs)
	for range subs {
		chans = append(chans, bc.Subscribe())
	}
	t.Cleanup(func() {
		for _, ch := range chans {
			bc.Unsubscribe(ch)
		}
	})

	bc.Broadcast(1) // fills every buffer
	bc.Broadcast(2) // buffer full for all 3 — drop fires 3 times

	if got := drops.Load(); got != subs {
		t.Errorf("expected %d drops (one per full subscriber), got %d", subs, got)
	}
}

func TestWithOnDrop_BroadcastMany(t *testing.T) {
	t.Parallel()

	var drops atomic.Int64

	bc := sse.NewBroadcaster[int](
		sse.WithBufferSize[int](1),
		sse.WithOnDrop[int](func(msg int) {
			drops.Add(1)
		}),
	)
	t.Cleanup(func() { bc.Close() })

	ch := bc.Subscribe()
	t.Cleanup(func() { bc.Unsubscribe(ch) })

	bc.BroadcastMany(1, 2, 3) // 1 fills buffer; 2 and 3 drop

	if got := drops.Load(); got != 2 {
		t.Errorf("expected 2 drops from BroadcastMany, got %d", got)
	}
}
