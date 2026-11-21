package sse_test

import (
	"runtime"
	"strconv"
	"testing"

	"github.com/larsartmann/go-sse"
)

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

			chans := make([]<-chan sse.Event, 0, subs)
			for range subs {
				chans = append(chans, bc.Subscribe())
			}

			runtime.GC()
			var after runtime.MemStats
			runtime.ReadMemStats(&after)

			_ = chans // keep subscribers alive across the measurement

			delta := after.HeapInuse - before.HeapInuse
			b.ReportMetric(float64(delta)/float64(subs), "B/sub")
			b.ReportMetric(float64(delta)/1024, "total-KiB")

			// No timed loop: this is a one-shot memory snapshot. b.N stays 1.
		})
	}
}
