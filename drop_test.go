package sse_test

import (
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-sse"
)

func TestWithOnDrop_FiresWhenBufferFull(t *testing.T) {
	t.Parallel()

	var drops atomic.Int64

	bc := sse.NewBroadcaster(
		sse.WithBufferSize[int](1), // buffer of 1 — second send drops
		sse.WithOnDrop(func(msg int) {
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

	bc := sse.NewBroadcaster(
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

	bc := sse.NewBroadcaster(
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

	bc := sse.NewBroadcaster(
		sse.WithBufferSize[int](1),
		sse.WithOnDrop(func(msg int) {
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

	bc := sse.NewBroadcaster(
		sse.WithBufferSize[int](1),
		sse.WithOnDrop(func(msg int) {
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

func TestWithOnDrop_ExplicitNilCallback(t *testing.T) {
	t.Parallel()

	bc := sse.NewBroadcaster(
		sse.WithBufferSize[int](1),
		sse.WithOnDrop[int](nil), // explicit nil — same contract as absent
	)
	t.Cleanup(func() { bc.Close() })

	ch := bc.Subscribe()
	t.Cleanup(func() { bc.Unsubscribe(ch) })

	// Must not panic even though an explicit nil callback is registered.
	bc.Broadcast(1) // fills the buffer
	bc.Broadcast(2) // buffer full — hits the nil-onDrop drop branch
}

func TestOnDrop_NilClearsCallback(t *testing.T) {
	t.Parallel()

	var drops atomic.Int64

	bc := sse.NewBroadcaster(sse.WithBufferSize[int](1))
	t.Cleanup(func() { bc.Close() })

	bc.OnDrop(func(msg int) {
		drops.Add(1)
	})
	bc.OnDrop(nil) // clear

	ch := bc.Subscribe()
	t.Cleanup(func() { bc.Unsubscribe(ch) })

	bc.Broadcast(1) // fills the buffer
	bc.Broadcast(2) // buffer full — drop branch, but no callback anymore

	if got := drops.Load(); got != 0 {
		t.Errorf("expected 0 drops after OnDrop(nil), got %d", got)
	}
}

func TestOnDrop_PanickingCallbackDoesNotCrashBroadcast(t *testing.T) {
	t.Parallel()

	var drops atomic.Int64

	bc := sse.NewBroadcaster(
		sse.WithBufferSize[int](1),
		sse.WithOnDrop(func(msg int) {
			drops.Add(1)
			panic("drop callback explosion")
		}),
	)
	t.Cleanup(func() { bc.Close() })

	ch := bc.Subscribe()
	t.Cleanup(func() { bc.Unsubscribe(ch) })

	bc.Broadcast(1) // fills the buffer
	bc.Broadcast(2) // drop → callback panics → must be recovered
	bc.Broadcast(3) // drop again — proves the fan-out loop continued past the panic

	if got := drops.Load(); got != 2 {
		t.Errorf("expected 2 (recovered) drops, got %d", got)
	}
}

func TestOnDrop_PanickingCallbackDoesNotCrashBroadcastMany(t *testing.T) {
	t.Parallel()

	var drops atomic.Int64

	bc := sse.NewBroadcaster(
		sse.WithBufferSize[int](1),
		sse.WithOnDrop(func(msg int) {
			drops.Add(1)
			panic("drop callback explosion")
		}),
	)
	t.Cleanup(func() { bc.Close() })

	ch := bc.Subscribe()
	t.Cleanup(func() { bc.Unsubscribe(ch) })

	bc.BroadcastMany(1, 2, 3, 4) // 1 fills; 2–4 drop, each firing a recovered panic

	if got := drops.Load(); got != 3 {
		t.Errorf("expected 3 (recovered) drops from BroadcastMany, got %d", got)
	}
}
