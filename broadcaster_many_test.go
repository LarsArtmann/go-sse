package sse_test

import (
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-sse"
)

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
