package sse_test

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	// matching events (minus drops during churn). Assert a meaningful threshold.
	if received.Load() < 500 {
		t.Errorf(
			"filtered subscriber received only %d matching events out of ~4000 sent",
			received.Load(),
		)
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

func TestSubscribeFilter_PredicatePanicRecovered(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	panicky := b.SubscribeFilter(func(evt sse.Event) bool {
		if evt.Data == "boom" {
			panic("predicate explosion")
		}

		return true
	})
	defer b.Unsubscribe(panicky)

	// A panicking predicate must NOT crash the broadcaster.
	// The panicking event is skipped (treated as non-match).
	b.Broadcast(sse.Event{Event: "safe", Data: "boom"})

	// After the panic, the broadcaster must still deliver subsequent events.
	b.Broadcast(sse.Event{Event: "safe", Data: "ok"})

	select {
	case evt := <-panicky:
		if evt.Data != "ok" {
			t.Errorf("expected ok after panic recovery, got %s", evt.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event after panic recovery")
	}
}

func TestSubscribeFilter_ShutdownDrainsFilteredSubscribers(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()

	filtered := b.SubscribeFilter(func(evt sse.Event) bool {
		return evt.Event == "keep"
	})

	// Broadcast matching + non-matching events.
	b.Broadcast(sse.Event{Event: "skip", Data: "nope"})
	b.Broadcast(sse.Event{Event: "keep", Data: "yes1"})
	b.Broadcast(sse.Event{Event: "skip", Data: "nope"})
	b.Broadcast(sse.Event{Event: "keep", Data: "yes2"})

	// Drain in a background goroutine while Shutdown waits for the buffer to empty.
	var got []string

	done := make(chan struct{})

	go func() {
		defer close(done)

		for evt := range filtered {
			got = append(got, evt.Data)
		}
	}()

	// Give the drain goroutine a moment to start consuming.
	time.Sleep(time.Millisecond)

	// Shutdown must drain the buffer — only matching events should arrive.
	if err := b.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	<-done

	want := []string{"yes1", "yes2"}
	if len(got) != len(want) {
		t.Fatalf("expected %d events, got %d: %v", len(want), len(got), got)
	}

	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d] expected %s, got %s", i, w, got[i])
		}
	}
}

// TestSubscribeFilter_DropPolicyRespected verifies that when a filtered
// subscriber's buffer is full, matching events are dropped (non-blocking) while
// non-matching events never enter the buffer. This confirms the filter+drop
// interaction: drops affect only matching events, and the predicate is evaluated
// before the send attempt.
func TestSubscribeFilter_DropPolicyRespected(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event](sse.WithBufferSize[sse.Event](2))

	ch := b.SubscribeFilter(func(evt sse.Event) bool {
		return evt.Event == "match"
	})
	defer b.Unsubscribe(ch)

	// Fill the 2-deep buffer with matching events (no consumer draining).
	for i := range 2 {
		b.Broadcast(sse.Event{Event: "match", Data: strconv.Itoa(i)})
	}

	// These matching events overflow the full buffer and are silently dropped.
	for i := 2; i < 10; i++ {
		b.Broadcast(sse.Event{Event: "match", Data: strconv.Itoa(i)})
	}

	// Non-matching events never enter the buffer regardless of fill state.
	for range 5 {
		b.Broadcast(sse.Event{Event: "skip", Data: "x"})
	}

	// Drain: only the first 2 matching events fit; the rest were dropped.
	var got []sse.Event

drain:
	for {
		select {
		case evt := <-ch:
			got = append(got, evt)
		default:
			break drain
		}
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 events (buffer capacity), got %d: %v", len(got), got)
	}

	for i, evt := range got {
		if evt.Event != "match" {
			t.Errorf("[%d] expected match, got %s", i, evt.Event)
		}

		if evt.Data != strconv.Itoa(i) {
			t.Errorf("[%d] expected %d, got %s", i, i, evt.Data)
		}
	}
}

// TestSubscribeFilter_BroadcastManyMixedSubscribers verifies correct event
// partition when half the subscribers have predicates and half do not, all
// receiving a single BroadcastMany batch. Each filtered subscriber must receive
// only matching events; each unfiltered subscriber must receive all events in
// order.
func TestSubscribeFilter_BroadcastManyMixedSubscribers(t *testing.T) {
	t.Parallel()

	b := sse.NewBroadcaster[sse.Event]()
	defer b.Close()

	const half = 5

	filtered := make([]<-chan sse.Event, half)
	unfiltered := make([]<-chan sse.Event, half)

	for i := range half {
		filtered[i] = b.SubscribeFilter(func(evt sse.Event) bool { return evt.Event == "keep" })
		unfiltered[i] = b.Subscribe()
	}

	b.BroadcastMany(
		sse.Event{Event: "keep", Data: "k1"},
		sse.Event{Event: "drop", Data: "d1"},
		sse.Event{Event: "keep", Data: "k2"},
	)

	for i, ch := range filtered {
		for j, want := range []string{"k1", "k2"} {
			evt := <-ch
			if evt.Data != want {
				t.Errorf("filtered[%d][%d]: expected %s, got %s", i, j, want, evt.Data)
			}
		}

		select {
		case extra := <-ch:
			t.Errorf("filtered[%d] should receive only matching events, got %v", i, extra)
		default:
		}
	}

	for i, ch := range unfiltered {
		for j, want := range []string{"k1", "d1", "k2"} {
			evt := <-ch
			if evt.Data != want {
				t.Errorf("unfiltered[%d][%d]: expected %s, got %s", i, j, want, evt.Data)
			}
		}
	}
}
