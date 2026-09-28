package ssetest

import (
	"io"
	"testing"

	"github.com/larsartmann/go-sse/sseparse"
)

// This file re-exports the parsing and assertion core from
// [github.com/larsartmann/go-sse/sseparse] so existing ssetest consumers keep
// compiling unchanged. The parser lives in its own zero-dependency module so
// non-HTTP consumers (unix-socket daemons, other SSE parsers wanting the WPT
// corpus as a spec oracle) can adopt it without pulling go-sse,
// go-branded-id, or go-error-family into their module graph. New code may
// import sseparse directly; every symbol here delegates to it.

// Event is an SSE event decoded from the wire format. See [sseparse.Event].
type Event = sseparse.Event

// ReadOption customizes how a reader scans the SSE wire format. See
// [sseparse.ReadOption].
type ReadOption = sseparse.ReadOption

// WithMaxLineBytes sets the cap on a single SSE wire line. See
// [sseparse.WithMaxLineBytes].
func WithMaxLineBytes(max int) ReadOption {
	return sseparse.WithMaxLineBytes(max)
}

// ReadEvents parses the SSE wire format from r and returns all decoded
// events. See [sseparse.ReadEvents] for the full wire-format semantics.
func ReadEvents(r io.Reader, opts ...ReadOption) ([]Event, error) {
	return sseparse.ReadEvents(r, opts...)
}

// ReadNEvents reads up to count events from r. See [sseparse.ReadNEvents].
func ReadNEvents(r io.Reader, count int, opts ...ReadOption) ([]Event, error) {
	return sseparse.ReadNEvents(r, count, opts...)
}

// MustReadEvents is like [ReadEvents] but calls tb.Fatal on error.
func MustReadEvents(tb testing.TB, r io.Reader, opts ...ReadOption) []Event {
	return sseparse.MustReadEvents(tb, r, opts...)
}

// MustReadNEvents is like [ReadNEvents] but calls tb.Fatal on error.
func MustReadNEvents(tb testing.TB, r io.Reader, count int, opts ...ReadOption) []Event {
	return sseparse.MustReadNEvents(tb, r, count, opts...)
}

// StreamReader reads SSE events from a live stream one at a time. See
// [sseparse.StreamReader].
type StreamReader = sseparse.StreamReader

// NewStreamReader creates a [StreamReader] for r. See
// [sseparse.NewStreamReader].
func NewStreamReader(r io.Reader, opts ...ReadOption) *StreamReader {
	return sseparse.NewStreamReader(r, opts...)
}

// MustReadNextEvent reads the next event from sr and calls tb.Fatal on error.
func MustReadNextEvent(tb testing.TB, sr *StreamReader) Event {
	return sseparse.MustReadNextEvent(tb, sr)
}

// FindByType returns the first event whose type matches, along with true. See
// [sseparse.FindByType].
func FindByType(events []Event, eventType string) (Event, bool) {
	return sseparse.FindByType(events, eventType)
}

// FilterByType returns only the events whose type matches. See
// [sseparse.FilterByType].
func FilterByType(events []Event, eventType string) []Event {
	return sseparse.FilterByType(events, eventType)
}

// EventsString returns a multi-line debug representation of an event slice.
// See [sseparse.EventsString].
func EventsString(events []Event) string {
	return sseparse.EventsString(events)
}

// RequireEventCount fails the test unless events has exactly want events. See
// [sseparse.RequireEventCount].
func RequireEventCount(tb testing.TB, events []Event, want int) {
	sseparse.RequireEventCount(tb, events, want)
}

// RequireEventType fails the test unless the event type matches want. See
// [sseparse.RequireEventType].
func RequireEventType(tb testing.TB, evt Event, want string) {
	sseparse.RequireEventType(tb, evt, want)
}

// RequireData fails the test unless the event payload exactly equals want. See
// [sseparse.RequireData].
func RequireData(tb testing.TB, evt Event, want string) {
	sseparse.RequireData(tb, evt, want)
}

// RequireDataContains fails the test unless the event payload contains
// wantContains as a substring. See [sseparse.RequireDataContains].
func RequireDataContains(tb testing.TB, evt Event, wantContains string) {
	sseparse.RequireDataContains(tb, evt, wantContains)
}

// RequireEventID fails the test unless the SSE event ID matches want. See
// [sseparse.RequireEventID].
func RequireEventID(tb testing.TB, evt Event, want string) {
	sseparse.RequireEventID(tb, evt, want)
}

// RequireRetry fails the test unless the event's reconnection interval
// matches want. See [sseparse.RequireRetry].
func RequireRetry(tb testing.TB, evt Event, want uint) {
	sseparse.RequireRetry(tb, evt, want)
}

// RequireDataJSON unmarshals the event payload as JSON into a fresh T and
// compares it to want with reflect.DeepEqual. See [sseparse.RequireDataJSON].
func RequireDataJSON[T any](tb testing.TB, evt Event, want T) {
	sseparse.RequireDataJSON(tb, evt, want)
}
