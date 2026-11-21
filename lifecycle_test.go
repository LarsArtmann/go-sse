package sse_test

import (
	"testing"

	"github.com/larsartmann/go-sse"
)

func TestBroadcaster_Health_InitialState(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	health := b.Health()
	if health.Closed {
		t.Error("fresh broadcaster should not be closed")
	}

	if health.Draining {
		t.Error("fresh broadcaster should not be draining")
	}

	if health.SubscriberCount != 0 {
		t.Errorf("SubscriberCount: got %d, want 0", health.SubscriberCount)
	}

	if health.BufferSize != sse.DefaultSubscriberBuffer {
		t.Errorf(
			"BufferSize: got %d, want default %d",
			health.BufferSize,
			sse.DefaultSubscriberBuffer,
		)
	}
}

func TestBroadcaster_Health_DuringOperation(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch1 := b.Subscribe()
	defer b.Unsubscribe(ch1)
	ch2 := b.Subscribe()
	defer b.Unsubscribe(ch2)

	health := b.Health()
	if health.SubscriberCount != 2 {
		t.Errorf("SubscriberCount: got %d, want 2", health.SubscriberCount)
	}

	if health.Closed || health.Draining {
		t.Error("operating broadcaster should not be closed or draining")
	}
}

func TestBroadcaster_Health_ReportsBufferSize(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster(sse.WithBufferSize[sse.Event](256))

	health := b.Health()
	if health.BufferSize != 256 {
		t.Errorf("BufferSize: got %d, want 256", health.BufferSize)
	}
}

func TestBroadcaster_Health_AfterClose(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	b.Close()

	health := b.Health()
	if !health.Closed {
		t.Error("Health.Closed should be true after Close")
	}
}

func TestBroadcaster_WithBufferSize_AppliesToNewSubscribers(t *testing.T) {
	t.Parallel()

	const size = 8
	b := sse.NewBroadcaster(sse.WithBufferSize[sse.Event](size))

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

	b := sse.NewBroadcaster(sse.WithBufferSize[sse.Event](0))

	if h := b.Health(); h.BufferSize != sse.DefaultSubscriberBuffer {
		t.Errorf(
			"BufferSize with 0 option: got %d, want default %d",
			h.BufferSize,
			sse.DefaultSubscriberBuffer,
		)
	}

	// Negative also ignored.
	b2 := sse.NewBroadcaster(sse.WithBufferSize[sse.Event](-1))
	if h := b2.Health(); h.BufferSize != sse.DefaultSubscriberBuffer {
		t.Errorf(
			"BufferSize with -1 option: got %d, want default %d",
			h.BufferSize,
			sse.DefaultSubscriberBuffer,
		)
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
