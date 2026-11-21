package sse_test

import (
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
	for range 10 {
		b.Broadcast(sse.Event{Event: "match", Data: "m"})
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

	const broadcastsPerGoroutine = 2000
	const numBroadcasters = 4
	// matchEvents = every other broadcast per goroutine is a "match"
	const matchEvents = numBroadcasters * broadcastsPerGoroutine / 2

	// The subscriber buffer holds the full burst so the non-blocking drop
	// policy can never fire for the persistent subscriber, no matter how the
	// race detector deschedules its collector goroutine. This turns the
	// received-count assertion into an exact, deterministic equality instead
	// of a flaky lower bound under CI contention.
	b := sse.NewBroadcaster(
		sse.WithBufferSize[sse.Event](matchEvents),
	)
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

	// No-loss: the buffer held the entire burst, so the subscriber must have
	// received every matching event — drops are only legal on a full buffer.
	if got := received.Load(); got != matchEvents {
		t.Errorf(
			"filtered subscriber received %d matching events, want exactly %d",
			got, matchEvents,
		)
	}
}
