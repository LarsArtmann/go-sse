package sse_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

func TestBroadcaster_Shutdown_Empty(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	if err := b.Shutdown(t.Context()); err != nil {
		t.Errorf("Shutdown on empty broadcaster: %v", err)
	}

	if !b.Health().Closed {
		t.Error("broadcaster should be closed after Shutdown")
	}
}

func TestBroadcaster_Shutdown_DrainsAllSubscribers(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	const numSubs = 4
	subs := make([]<-chan sse.Event, 0, numSubs)
	for range numSubs {
		subs = append(subs, b.Subscribe())
	}

	// Push 3 events; each subscriber's buffer holds 3.
	for i := range 3 {
		b.Broadcast(sse.Event{Event: "drain", Data: string(rune('a' + i))})
	}

	// Drain in a background goroutine: each subscriber reads 3 events.
	for _, ch := range subs {
		go func(ch <-chan sse.Event) {
			for range 3 {
				<-ch
			}
		}(ch)
	}

	// Give the drain goroutines a moment to start consuming so the
	// buffer is empty when Shutdown begins polling.
	time.Sleep(time.Millisecond)

	if err := b.Shutdown(t.Context()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}

	// All subscriber channels should be closed after a successful drain.
	for i, ch := range subs {
		if _, ok := <-ch; ok {
			t.Errorf("sub %d: channel should be closed after Shutdown", i)
		}
	}

	if !b.Health().Closed {
		t.Error("Health.Closed should be true after Shutdown")
	}
}

func TestBroadcaster_Shutdown_ContextCancel(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	ch := b.Subscribe()
	defer func() {
		// Clean up: instant close after the test asserts the cancelled state.
		b.Close()
	}()

	// Push one event; never drain it.
	b.Broadcast(sse.Event{Event: "stuck", Data: "x"})

	// Pre-cancel a context so Shutdown returns immediately with the
	// cancellation error. The drain loop checks the context after the
	// first tick (~1ms), so a cancelled context returns within the
	// poll interval.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := b.Shutdown(ctx); err == nil {
		t.Fatal("Shutdown should return context error on cancel")
	}

	// Broadcaster must remain in draining state, NOT closed. The
	// caller can retry with a fresh context or fall back to Close.
	h := b.Health()
	if h.Closed {
		t.Error("broadcaster should NOT be closed when Shutdown's ctx fires")
	}

	if !h.Draining {
		t.Error("broadcaster should be in draining state after cancelled Shutdown")
	}

	if h.SubscriberCount != 1 {
		t.Errorf("SubscriberCount: got %d, want 1", h.SubscriberCount)
	}

	// ch should still be open (not closed by the cancelled Shutdown).
	select {
	case _, ok := <-ch:
		if !ok {
			t.Error("subscriber channel should still be open after cancelled Shutdown")
		}
	default:
		// Channel still open and may or may not have a buffered value.
	}
}

func TestBroadcaster_Shutdown_RejectsNewSubscribersWhileDraining(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	// One subscriber that never drains.
	ch1 := b.Subscribe()

	// Push an event so the drain loop has something to wait for.
	b.Broadcast(sse.Event{Event: "stuck", Data: "x"})

	// Cancel the shutdown context quickly.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_ = b.Shutdown(ctx) // returns ctx.Err()

	// New Subscribe during draining should return a closed channel.
	ch2 := b.Subscribe()
	if _, ok := <-ch2; ok {
		t.Error("Subscribe during draining should return a closed channel")
	}

	// Cleanup
	b.Close()
	_ = ch1
}

func TestBroadcaster_Shutdown_Idempotent(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	if err := b.Shutdown(t.Context()); err != nil {
		t.Fatalf("first Shutdown: %v", err)
	}

	// Second Shutdown on a closed broadcaster should be a no-op.
	if err := b.Shutdown(t.Context()); err != nil {
		t.Errorf("second Shutdown on closed broadcaster: %v", err)
	}
}

func TestBroadcaster_Shutdown_AfterCloseIsNoop(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	b.Close()

	if err := b.Shutdown(t.Context()); err != nil {
		t.Errorf("Shutdown after Close: %v", err)
	}
}

func TestBroadcaster_Health_InitialState(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	h := b.Health()
	if h.Closed {
		t.Error("fresh broadcaster should not be closed")
	}

	if h.Draining {
		t.Error("fresh broadcaster should not be draining")
	}

	if h.SubscriberCount != 0 {
		t.Errorf("SubscriberCount: got %d, want 0", h.SubscriberCount)
	}

	if h.BufferSize != 64 {
		t.Errorf("BufferSize: got %d, want default 64", h.BufferSize)
	}
}

func TestBroadcaster_Health_DuringOperation(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch1 := b.Subscribe()
	defer b.Unsubscribe(ch1)
	ch2 := b.Subscribe()
	defer b.Unsubscribe(ch2)

	h := b.Health()
	if h.SubscriberCount != 2 {
		t.Errorf("SubscriberCount: got %d, want 2", h.SubscriberCount)
	}

	if h.Closed || h.Draining {
		t.Error("operating broadcaster should not be closed or draining")
	}
}

func TestBroadcaster_Health_ReportsBufferSize(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event](sse.WithBufferSize[sse.Event](256))

	h := b.Health()
	if h.BufferSize != 256 {
		t.Errorf("BufferSize: got %d, want 256", h.BufferSize)
	}
}

func TestBroadcaster_Health_AfterClose(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	b.Close()

	h := b.Health()
	if !h.Closed {
		t.Error("Health.Closed should be true after Close")
	}
}

func TestBroadcaster_WithBufferSize_AppliesToNewSubscribers(t *testing.T) {
	t.Parallel()

	const size = 8
	b := sse.NewBroadcaster[sse.Event](sse.WithBufferSize[sse.Event](size))

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// Send size+1 events; the buffer holds size, the (size+1)th is dropped.
	for i := range size + 1 {
		b.Broadcast(sse.Event{Event: "e", Data: string(rune('a' + i))})
	}

	received := 0

drain:
	for {
		select {
		case <-ch:
			received++
		default:
			break drain
		}
	}

	if received != size {
		t.Errorf("expected %d (buffer size), got %d", size, received)
	}
}

func TestBroadcaster_WithBufferSize_NonPositiveIsIgnored(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event](sse.WithBufferSize[sse.Event](0))

	if h := b.Health(); h.BufferSize != 64 {
		t.Errorf("BufferSize with 0 option: got %d, want default 64", h.BufferSize)
	}

	// Negative also ignored.
	b2 := sse.NewBroadcaster[sse.Event](sse.WithBufferSize[sse.Event](-1))
	if h := b2.Health(); h.BufferSize != 64 {
		t.Errorf("BufferSize with -1 option: got %d, want default 64", h.BufferSize)
	}
}

func TestBroadcaster_Shutdown_ConcurrentWithUnsubscribe(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	const numSubs = 16
	subs := make([]<-chan sse.Event, 0, numSubs)
	for range numSubs {
		subs = append(subs, b.Subscribe())
	}

	// Half the subscribers disconnect during the drain; the rest
	// get drained by a background reader. Shutdown must tolerate both.
	for i := range numSubs / 2 {
		b.Unsubscribe(subs[i])
	}

	for i := numSubs / 2; i < numSubs; i++ {
		go func(ch <-chan sse.Event) {
			// Read whatever was buffered, then return. Unsubscribe will
			// close the channel and we'll see it as a closed receive.
			for range 10 {
				_, ok := <-ch
				if !ok {
					return
				}
			}
		}(subs[i])
	}

	if err := b.Shutdown(t.Context()); err != nil {
		t.Errorf("Shutdown with concurrent unsubscribes: %v", err)
	}

	if !b.Health().Closed {
		t.Error("broadcaster should be closed after successful Shutdown")
	}
}
