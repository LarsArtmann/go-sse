package sse_test

import (
	"runtime"
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

func TestBroadcaster_BroadcastMany(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch1 := b.Subscribe()
	ch2 := b.Subscribe()
	defer b.Unsubscribe(ch1)
	defer b.Unsubscribe(ch2)

	events := []sse.Event{
		{Event: "a", Data: "1"},
		{Event: "b", Data: "2"},
		{Event: "c", Data: "3"},
	}
	b.BroadcastMany(events...)

	for i, want := range events {
		if got := <-ch1; got != want {
			t.Errorf("ch1[%d]: got %v, want %v", i, got, want)
		}

		if got := <-ch2; got != want {
			t.Errorf("ch2[%d]: got %v, want %v", i, got, want)
		}
	}
}

func TestBroadcaster_BroadcastMany_PreservesOrder(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// Stay within the 64-deep subscriber buffer so all events are delivered
	// (BroadcastMany is non-blocking: events beyond the buffer are dropped).
	const n = 50
	events := make([]sse.Event, 0, n)
	for i := range n {
		events = append(events, sse.Event{Data: strconv.Itoa(i)})
	}

	b.BroadcastMany(events...)

	for i, want := range events {
		got := <-ch
		if got != want {
			t.Errorf("event %d: got %v, want %v", i, got, want)
		}
	}
}

func TestBroadcaster_BroadcastMany_Empty(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.BroadcastMany() // no-op, must not panic or block

	if b.SubscriberCount() != 1 {
		t.Errorf("subscriber count: got %d, want 1", b.SubscriberCount())
	}
}

func TestBroadcaster_BroadcastMany_MixedSlowFast(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	fastCh := b.Subscribe()
	slowCh := b.Subscribe()

	const (
		numBatches  = 20
		batchSize   = 10
		totalEvents = numBatches * batchSize // 200, well above the 64-deep buffer
	)

	var fastCount atomic.Int64

	// Synchronization channel: the drain goroutine signals after each receive.
	// Capacity equals totalEvents so no signal is lost between batches.
	drained := make(chan struct{}, totalEvents)

	go func() {
		for range fastCh {
			fastCount.Add(1)
			drained <- struct{}{}
		}
	}()

	// Send events in batches, deterministically waiting for the drain goroutine
	// to process each batch before sending the next. No time.Sleep — the
	// channel synchronization guarantees the fast subscriber keeps up.
	for batch := range numBatches {
		events := make([]sse.Event, 0, batchSize)
		for i := range batchSize {
			events = append(events, sse.Event{Data: strconv.Itoa(batch*batchSize + i)})
		}

		b.BroadcastMany(events...)

		for range batchSize {
			<-drained
		}
	}

	// Slow subscriber was never drained: buffer capped at exactly 64.
	slowCount := 0

drainSlow:
	for {
		select {
		case <-slowCh:
			slowCount++
		default:
			break drainSlow
		}
	}

	b.Close() // closes fastCh → drain goroutine exits

	if slowCount != 64 {
		t.Errorf("slow subscriber: got %d events, want exactly 64 (buffer size)", slowCount)
	}

	if fast := int(fastCount.Load()); fast != totalEvents {
		t.Errorf("fast subscriber: got %d events, want %d", fast, totalEvents)
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
	for _, subs := range []int{1, 10, 100, 1000, 10000} {
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

// BenchmarkBroadcastManyVsLoop compares a single BroadcastMany(100) call against
// 100 individual Broadcast calls, demonstrating the single-lock-acquisition
// advantage of the batch API.
func BenchmarkBroadcastManyVsLoop(b *testing.B) {
	const subs = 100

	events := make([]sse.Event, 0, 100)
	for range 100 {
		events = append(events, sse.Event{Event: "batch", Data: "payload"})
	}

	b.Run("BroadcastMany", func(b *testing.B) {
		bc := sse.NewBroadcaster[sse.Event]()
		defer bc.Close()

		for range subs {
			drain(b, bc.Subscribe())
		}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			bc.BroadcastMany(events...)
		}
	})

	b.Run("BroadcastLoop", func(b *testing.B) {
		bc := sse.NewBroadcaster[sse.Event]()
		defer bc.Close()

		for range subs {
			drain(b, bc.Subscribe())
		}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			for _, evt := range events {
				bc.Broadcast(evt)
			}
		}
	})
}

// BenchmarkMemoryPerSubscriber measures steady-state heap memory per subscriber
// at scale. Each subscriber owns a buffered channel (capacity 64 by default) of
// Events, plus the subscriber struct and its map entry. This isolates the
// per-connection memory floor independent of event throughput.
//
// The buffer dominates: 64 × sizeof(sse.Event) ≈ 64 × 56 = 3.5 KiB, so the
// per-subscriber cost is ~3.6–3.8 KiB regardless of how many events flow.
func BenchmarkMemoryPerSubscriber(b *testing.B) {
	for _, subs := range []int{100, 1000, 10000} {
		b.Run(strconv.Itoa(subs)+"subs", func(b *testing.B) {
			bc := sse.NewBroadcaster[sse.Event]()
			defer bc.Close()

			runtime.GC()
			var before runtime.MemStats
			runtime.ReadMemStats(&before)

			chans := make([]<-chan sse.Event, subs)
			for i := range subs {
				chans[i] = bc.Subscribe()
			}

			runtime.GC()
			var after runtime.MemStats
			runtime.ReadMemStats(&after)

			_ = chans // keep subscribers alive across the measurement

			delta := int64(after.HeapInuse) - int64(before.HeapInuse)
			b.ReportMetric(float64(delta)/float64(subs), "B/sub")
			b.ReportMetric(float64(delta)/1024, "total-KiB")

			// No timed loop: this is a one-shot memory snapshot. b.N stays 1.
		})
	}
}
