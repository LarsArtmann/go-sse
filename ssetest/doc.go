// Package ssetest provides helpers for E2E testing Server-Sent Events
// handlers built on go-sse — without hand-rolling an SSE parser.
//
// It solves the two problems that make SSE handlers hard to test:
//
//  1. Parsing the SSE wire format (event:/data:/id:/retry: lines) into events.
//  2. Driving a real HTTP handler end-to-end and asserting on what it sent.
//
// The package is a separate Go module (it depends on testing), so it never
// leaks into production builds of go-sse consumers.
//
// # Quick start
//
//	import (
//	    "net/http"
//	    "testing"
//
//	    "github.com/larsartmann/go-sse"
//	    "github.com/larsartmann/go-sse/ssetest"
//	)
//
//	func TestFeedHandler(t *testing.T) {
//	    t.Parallel()
//
//	    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        stream := sse.NewStream(w, r)
//	        defer func() { _ = stream.Close() }()
//
//	        _ = stream.Send(sse.Event{Event: "feed", Data: "hello"})
//	    })
//
//	    events := ssetest.Collect(t, handler)
//	    ssetest.RequireEventCount(t, events, 1)
//	    ssetest.RequireEventType(t, events[0], "feed")
//	    ssetest.RequireData(t, events[0], "hello")
//	}
//
// # Request options
//
// Every Collect helper accepts [RequestOption]s: [WithPath] targets a route on
// a mux (query strings allowed), [WithLastEventID] simulates a reconnecting
// browser for replay testing, and [WithHeader] adds any custom header:
//
//	events := ssetest.Collect(t, mux, ssetest.WithPath("/events?filter=alerts"))
//	events := ssetest.Collect(t, handler, ssetest.WithLastEventID("42"))
//
// # Streaming handlers
//
// For handlers that keep the connection open (e.g., broadcasting through a
// [sse.Broadcaster]), use [CollectN] to read exactly N events then close.
// Use [CollectWithTimeout] for a time-bounded read that returns whatever
// events arrived before the deadline:
//
//	events := ssetest.CollectN(t, handler, 3)
//	events := ssetest.CollectWithTimeout(t, handler, 5*time.Second)
//
// # Replay testing
//
// Combine [WithLastEventID] with [RequireEventID] to E2E test the full
// reconnection story: the handler replays missed events from its
// [sse.EventStore], and the test drives the reconnect like a browser would.
//
// # Ginkgo and benchmarks
//
// All helpers that take a `t` accept [testing.TB], not *testing.T, so they
// work with *testing.T, *testing.B, and Ginkgo's GinkgoT().
//
// # Debugging
//
// Use [Event.String] and [EventsString] for human-readable representations
// useful in test failure messages:
//
//	t.Fatalf("unexpected events:\n%s", ssetest.EventsString(events))
package ssetest
