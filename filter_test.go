package sse_test

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-sse"
)

func TestSubscribeFilter_DeliversOnlyMatching(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	matching := b.SubscribeFilter(func(evt sse.Event) bool {
		return evt.Event == "match"
	})
	defer b.Unsubscribe(matching)

	b.Broadcast(sse.Event{Event: "match", Data: "yes"})
	b.Broadcast(sse.Event{Event: "skip", Data: "no"})
	b.Broadcast(sse.Event{Event: "match", Data: "yes2"})

	got1 := <-matching
	if got1.Data != "yes" {
		t.Fatalf("first: expected yes, got %s", got1.Data)
	}

	got2 := <-matching
	if got2.Data != "yes2" {
		t.Fatalf("second: expected yes2, got %s", got2.Data)
	}
}

func TestSubscribeFilter_NilPredicateEqualsSubscribe(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch := b.SubscribeFilter(nil)
	defer b.Unsubscribe(ch)

	if b.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber, got %d", b.SubscriberCount())
	}

	evt := sse.Event{Event: "update", Data: "hello"}
	b.Broadcast(evt)

	if got := <-ch; got != evt {
		t.Errorf("nil pred should deliver all; got %v", got)
	}
}

func TestSubscribeFilter_MixedFilteredAndUnfiltered(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	all := b.Subscribe()
	defer b.Unsubscribe(all)

	onlyMsg := b.SubscribeFilter(func(evt sse.Event) bool {
		return evt.Event == "message"
	})
	defer b.Unsubscribe(onlyMsg)

	b.Broadcast(sse.Event{Event: "message", Data: "msg1"})
	b.Broadcast(sse.Event{Event: "reaction", Data: "react1"})

	// Unfiltered subscriber gets both
	if got := <-all; got.Event != "message" {
		t.Errorf("all: expected message, got %s", got.Event)
	}

	if got := <-all; got.Event != "reaction" {
		t.Errorf("all: expected reaction, got %s", got.Event)
	}

	// Filtered subscriber gets only message
	if got := <-onlyMsg; got.Event != "message" {
		t.Errorf("filtered: expected message, got %s", got.Event)
	}

	select {
	case extra := <-onlyMsg:
		t.Errorf("filtered should not have received reaction, got %v", extra)
	default:
	}
}

func TestSubscribeFilter_AfterCloseReturnsClosedChannel(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	b.Close()

	ch := b.SubscribeFilter(func(evt sse.Event) bool { return true })

	if _, ok := <-ch; ok {
		t.Error("channel should be closed after Close + SubscribeFilter")
	}
}

func TestSubscribeFilter_BufferOnlyFillsWithMatching(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	ch := b.SubscribeFilter(func(evt sse.Event) bool {
		return evt.Event == "match"
	})
	defer b.Unsubscribe(ch)

	// Broadcast 100 non-matching events — none should enter the buffer
	for range 100 {
		b.Broadcast(sse.Event{Event: "skip", Data: "x"})
	}

	// Broadcast 10 matching events
	for i := range 10 {
		b.Broadcast(sse.Event{Event: "match", Data: "m"})
		_ = i
	}

	// Should receive all 10 matching events
	count := 0
	for range 10 {
		<-ch
		count++
	}

	if count != 10 {
		t.Errorf("expected 10 matching events, got %d", count)
	}
}

func TestSubscribeFilter_BroadcastManyRespectsPredicates(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	onlyMsg := b.SubscribeFilter(func(evt sse.Event) bool {
		return evt.Event == "message"
	})
	defer b.Unsubscribe(onlyMsg)

	all := b.Subscribe()
	defer b.Unsubscribe(all)

	b.BroadcastMany(
		sse.Event{Event: "message", Data: "msg1"},
		sse.Event{Event: "reaction", Data: "react1"},
		sse.Event{Event: "message", Data: "msg2"},
		sse.Event{Event: "reaction", Data: "react2"},
	)

	// Unfiltered subscriber gets all 4 in order
	for i, want := range []string{"msg1", "react1", "msg2", "react2"} {
		evt := <-all
		if evt.Data != want {
			t.Errorf("unfiltered[%d]: expected %s, got %s", i, want, evt.Data)
		}
	}

	// Filtered subscriber gets only the 2 "message" events in order
	got := <-onlyMsg
	if got.Data != "msg1" {
		t.Fatalf("filtered first: expected msg1, got %s", got.Data)
	}

	got = <-onlyMsg
	if got.Data != "msg2" {
		t.Fatalf("filtered second: expected msg2, got %s", got.Data)
	}

	select {
	case extra := <-onlyMsg:
		t.Errorf("filtered should not receive more after BroadcastMany, got %v", extra)
	default:
	}
}

func TestSubscribeFilter_ConcurrentRace(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	// Persistent filtered subscriber: only "match" events
	filtered := b.SubscribeFilter(func(evt sse.Event) bool {
		return evt.Event == "match"
	})

	// Collector goroutine: drain and verify every received event is "match"
	var received atomic.Int64
	var wrong atomic.Int64

	done := make(chan struct{})

	go func() {
		defer close(done)

		for evt := range filtered {
			received.Add(1)
			if evt.Event != "match" {
				wrong.Add(1)
			}
		}
	}()

	const broadcastsPerGoroutine = 2000
	const numBroadcasters = 4

	var wg sync.WaitGroup

	// Concurrent broadcasters: mix of match and skip
	for range numBroadcasters {
		wg.Go(func() {
			for i := range broadcastsPerGoroutine {
				if i%2 == 0 {
					b.Broadcast(sse.Event{Event: "match", Data: "x"})
				} else {
					b.Broadcast(sse.Event{Event: "skip", Data: "x"})
				}
			}
		})
	}

	// Subscriber churn: subscribe/unsubscribe with filters concurrently
	wg.Go(func() {
		for range 500 {
			ch := b.SubscribeFilter(func(evt sse.Event) bool {
				return evt.Event == "match"
			})
			b.Unsubscribe(ch)
		}
	})

	wg.Wait()

	// Close the persistent subscriber and wait for the collector to finish
	b.Unsubscribe(filtered)
	<-done

	// Correctness: every received event must be "match"
	if wrong.Load() > 0 {
		t.Errorf("filtered subscriber received %d non-matching events out of %d total",
			wrong.Load(), received.Load())
	}

	// Sanity: with 4 broadcasters × 1000 "match" events each, we expect ~4000
	if received.Load() == 0 {
		t.Error("filtered subscriber should have received at least some events")
	}
}

// BenchmarkSubscribeFilter_PredicateOverhead measures the cost of the predicate
// function call in the broadcast hot path. Compares unfiltered (nil predicate,
// branch-not-taken) against filtered (always-true predicate, function called)
// at 1, 100, and 1000 subscribers to isolate per-subscriber overhead.
func BenchmarkSubscribeFilter_PredicateOverhead(b *testing.B) {
	evt := sse.Event{Event: "bench", Data: "payload"}

	for _, subs := range []int{1, 100, 1000} {
		b.Run(strconv.Itoa(subs)+"subs/unfiltered", func(b *testing.B) {
			bc := sse.NewBroadcaster[sse.Event]()
			defer bc.Close()

			for range subs {
				drain(b, bc.Subscribe())
			}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				bc.Broadcast(evt)
			}
		})

		b.Run(strconv.Itoa(subs)+"subs/filtered", func(b *testing.B) {
			bc := sse.NewBroadcaster[sse.Event]()
			defer bc.Close()

			pred := func(sse.Event) bool { return true }

			for range subs {
				drain(b, bc.SubscribeFilter(pred))
			}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				bc.Broadcast(evt)
			}
		})
	}
}
