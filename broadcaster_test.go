package sse_test

import (
	"strconv"
	"sync"
	"sync/atomic"
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

func BenchmarkBroadcasterFanOut(b *testing.B) {
	for _, subs := range []int{1, 10, 100, 1000} {
		b.Run(strconv.Itoa(subs), func(b *testing.B) {
			bc := sse.NewBroadcaster[sse.Event]()
			defer bc.Close()

			for range subs {
				ch := bc.Subscribe()
				drain(b, ch)
			}

			evt := sse.Event{Event: "bench", Data: "payload"}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				bc.Broadcast(evt)
			}
		})
	}
}

func BenchmarkSubscribeUnsubscribe(b *testing.B) {
	bc := sse.NewBroadcaster[sse.Event]()
	defer bc.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		ch := bc.Subscribe()
		bc.Unsubscribe(ch)
	}
}
