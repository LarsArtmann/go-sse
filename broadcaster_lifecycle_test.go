package sse_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-sse"
)

func TestBroadcaster_OnSubscribeHook(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	count := 0

	b.OnSubscribe(func() { count++ })

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	ch2 := b.Subscribe()
	defer b.Unsubscribe(ch2)

	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestBroadcaster_OnUnsubscribeHook(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	count := 0

	b.OnUnsubscribe(func() { count++ })

	ch := b.Subscribe()
	b.Unsubscribe(ch)

	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestBroadcaster_OnUnsubscribeNotFiredForUnknown(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	count := 0

	b.OnUnsubscribe(func() { count++ })

	unknown := make(chan sse.Event)
	b.Unsubscribe(unknown)

	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestBroadcaster_ConcurrentHookCount(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	var subCount, unsubCount atomic.Int64

	b.OnSubscribe(func() { subCount.Add(1) })
	b.OnUnsubscribe(func() { unsubCount.Add(1) })

	var wg sync.WaitGroup

	const goroutines = 20
	const cycles = 100

	for range goroutines {
		wg.Go(func() {
			for range cycles {
				ch := b.Subscribe()
				b.Unsubscribe(ch)
			}
		})
	}

	wg.Wait()

	total := int64(goroutines * cycles)
	if subCount.Load() != total {
		t.Errorf("subCount: got %d, want %d", subCount.Load(), total)
	}

	if unsubCount.Load() != total {
		t.Errorf("unsubCount: got %d, want %d", unsubCount.Load(), total)
	}
}

func TestBroadcaster_Close(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch1 := b.Subscribe()
	ch2 := b.Subscribe()

	b.Close()

	if _, ok := <-ch1; ok {
		t.Error("ch1 should be closed")
	}

	if _, ok := <-ch2; ok {
		t.Error("ch2 should be closed")
	}
}

func TestBroadcaster_SubscribeAfterClose(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	b.Close()

	ch := b.Subscribe()

	if _, ok := <-ch; ok {
		t.Error("channel should be closed after Close + Subscribe")
	}
}

func TestBroadcaster_BroadcastAfterClose(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	b.Close()

	b.Broadcast(sse.Event{Event: "after-close", Data: "x"}) // must not panic
}
