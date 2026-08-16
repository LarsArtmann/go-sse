package ssetest

import (
	"strings"
	"testing"
)

// All Require* helpers accept [testing.TB] rather than *testing.T, so they work
// with *testing.T, *testing.B, and Ginkgo's GinkgoT().

// RequireEventCount fails the test unless events has exactly want events.
func RequireEventCount(tb testing.TB, events []Event, want int) {
	tb.Helper()

	if len(events) != want {
		tb.Fatalf("event count: got %d, want %d", len(events), want)
	}
}

// RequireEventType fails the test unless the event type matches want.
func RequireEventType(tb testing.TB, evt Event, want string) {
	tb.Helper()

	if evt.Type != want {
		tb.Fatalf("event type: got %q, want %q", evt.Type, want)
	}
}

// RequireData fails the test unless the event payload (data lines rejoined
// by "\n") exactly equals want.
func RequireData(tb testing.TB, evt Event, want string) {
	tb.Helper()

	if got := evt.Data(); got != want {
		tb.Errorf("data: got %q, want %q", got, want)
	}
}

// RequireDataContains fails the test unless the event payload contains
// wantContains as a substring. Useful for large payloads where an exact match
// is brittle.
func RequireDataContains(tb testing.TB, evt Event, wantContains string) {
	tb.Helper()

	if got := evt.Data(); !strings.Contains(got, wantContains) {
		tb.Errorf("data should contain %q; got %q", wantContains, got)
	}
}

// RequireEventID fails the test unless the SSE event ID matches want. Use this
// to verify that replayable handlers assign the expected event IDs (e.g., when
// testing reconnection replay driven by [WithLastEventID]).
func RequireEventID(tb testing.TB, evt Event, want string) {
	tb.Helper()

	if evt.ID != want {
		tb.Fatalf("event ID: got %q, want %q", evt.ID, want)
	}
}

// RequireRetry fails the test unless the event's reconnection interval (the
// retry: field, in milliseconds) matches want. A zero want asserts the field
// was not emitted.
func RequireRetry(tb testing.TB, evt Event, want uint) {
	tb.Helper()

	if evt.Retry != want {
		tb.Fatalf("retry: got %d, want %d", evt.Retry, want)
	}
}
