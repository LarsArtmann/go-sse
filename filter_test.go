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

func TestSubscribeFilter_ConcurrentRace(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	var stop atomic.Bool

	var wg sync.WaitGroup

	// Broadcasters: mix of matching and non-matching events
	for range 4 {
		wg.Go(func() {
			for i := 0; !stop.Load(); i++ {
				if i%2 == 0 {
					b.Broadcast(sse.Event{Event: "match", Data: "x"})
				} else {
					b.Broadcast(sse.Event{Event: "skip", Data: "x"})
				}
			}
		})
	}

	// Subscriber churn: subscribe/unsubscribe with filters
	wg.Go(func() {
		for !stop.Load() {
			ch := b.SubscribeFilter(func(evt sse.Event) bool {
				return evt.Event == "match"
			})
			b.Unsubscribe(ch)
		}
	})

	// Let it run briefly
	stop.Store(true)
	wg.Wait()
}
