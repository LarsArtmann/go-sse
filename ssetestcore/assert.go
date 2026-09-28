package ssetest

import (
	"encoding/json/v2"
	"reflect"
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

// RequireDataJSON unmarshals the event payload as JSON into a fresh T and
// compares it to want with reflect.DeepEqual. Use it for JSON-payload handlers
// (e.g. DataStar signals) instead of hand-rolled unmarshal-and-compare blocks:
// the comparison is structural, so key order and whitespace in the payload do
// not matter.
//
//	ssetest.RequireDataJSON(tb, evt, map[string]any{"progress": 50.0})
//	// or with a typed want:
//	ssetest.RequireDataJSON(tb, evt, mySignal{Progress: 50})
//
// The type parameter is inferred from want. A payload that fails to unmarshal
// fails the test immediately (Fatal), with the raw payload in the message.
func RequireDataJSON[T any](tb testing.TB, evt Event, want T) {
	tb.Helper()

	var got T

	payload := evt.Data()
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		tb.Fatalf("unmarshal event data as %T: %v\npayload: %q", want, err, payload)
	}

	if !reflect.DeepEqual(got, want) {
		tb.Errorf("data json: got %#v, want %#v\npayload: %q", got, want, payload)
	}
}
