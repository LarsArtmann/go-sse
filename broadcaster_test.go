package sse_test

import (
	"sync"
	"testing"

	"github.com/larsartmann/go-sse"
)

func TestBroadcaster_StartsEmpty(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	if b.SubscriberCount() != 0 {
		t.Fatalf("expected 0, got %d", b.SubscriberCount())
	}
}

func TestBroadcaster_Subscribe(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	if b.SubscriberCount() != 1 {
		t.Errorf("expected 1, got %d", b.SubscriberCount())
	}
}

func TestBroadcaster_MultipleSubscribers(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch1 := b.Subscribe()
	ch2 := b.Subscribe()
	ch3 := b.Subscribe()

	if b.SubscriberCount() != 3 {
		t.Fatalf("expected 3, got %d", b.SubscriberCount())
	}

	b.Unsubscribe(ch1)
	b.Unsubscribe(ch2)
	b.Unsubscribe(ch3)

	if b.SubscriberCount() != 0 {
		t.Errorf("expected 0 after all unsubscribe, got %d", b.SubscriberCount())
	}
}

func TestBroadcaster_BroadcastDelivers(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch1 := b.Subscribe()
	ch2 := b.Subscribe()
	defer b.Unsubscribe(ch1)
	defer b.Unsubscribe(ch2)

	evt := sse.Event{Event: "update", Data: "<div>new</div>"}
	b.Broadcast(evt)

	if got := <-ch1; got != evt {
		t.Errorf("ch1: got %v", got)
	}

	if got := <-ch2; got != evt {
		t.Errorf("ch2: got %v", got)
	}
}

func TestBroadcaster_UnsubscribeClosesChannel(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch := b.Subscribe()
	b.Unsubscribe(ch)

	if _, ok := <-ch; ok {
		t.Error("channel should be closed")
	}
}

func TestBroadcaster_UnsubscribeUnknownIsSafe(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	unknown := make(chan sse.Event)

	b.Unsubscribe(unknown) // should not panic

	if b.SubscriberCount() != 0 {
		t.Errorf("expected 0, got %d", b.SubscriberCount())
	}
}

func TestBroadcaster_DropsOnFullBuffer(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// Buffer is 64; send 65 to overflow
	for i := range 65 {
		b.Broadcast(sse.Event{Event: "e", Data: string(rune(i))})
	}

	received := 0

	for {
		select {
		case <-ch:
			received++
		default:
			goto done
		}
	}

done:
	if received != 64 {
		t.Errorf("expected 64 (buffer size), got %d", received)
	}
}

func TestBroadcaster_ConcurrentSafety(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	var wg sync.WaitGroup
	const goroutines = 10

	for range goroutines {
		ch := b.Subscribe()

		wg.Go(func() {
			defer b.Unsubscribe(ch)

			<-ch
		})
	}

	b.Broadcast(sse.Event{Event: "test", Data: "concurrent"})

	wg.Wait()

	if b.SubscriberCount() != 0 {
		t.Errorf("expected 0 after all goroutines, got %d", b.SubscriberCount())
	}
}

func TestBroadcaster_BroadcastUnsubscribeRace(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	stop := make(chan struct{})

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					b.Broadcast(sse.Event{Event: "hammer", Data: "x"})
				}
			}
		})
	}

	wg.Go(func() {
		defer close(stop)

		for range 2000 {
			ch := b.Subscribe()
			b.Unsubscribe(ch)
		}
	})

	wg.Wait()
}

func TestBroadcaster_InOrderDelivery(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	for i := range 5 {
		b.Broadcast(sse.Event{Event: "seq", Data: string(rune('0' + i))})
	}

	for i := range 5 {
		evt := <-ch
		want := string(rune('0' + i))
		if evt.Data != want {
			t.Errorf("event %d: got %q, want %q", i, evt.Data, want)
		}
	}
}

func TestBroadcaster_GenericWithString(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[string]()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Broadcast("hello")

	if got := <-ch; got != "hello" {
		t.Errorf("got %q", got)
	}
}

// --- Benchmarks ---

func drain[T any](tb testing.TB, ch <-chan T) {
	tb.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range ch {
		}
	}()

	tb.Cleanup(func() { <-done })
}
