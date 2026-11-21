package sse_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

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

	b := sse.NewBroadcaster(sse.WithBufferSize[sse.Event](2))

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

	filtered := make([]<-chan sse.Event, 0, half)
	unfiltered := make([]<-chan sse.Event, 0, half)

	for range half {
		filtered = append(
			filtered,
			b.SubscribeFilter(func(evt sse.Event) bool { return evt.Event == "keep" }),
		)
		unfiltered = append(unfiltered, b.Subscribe())
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
