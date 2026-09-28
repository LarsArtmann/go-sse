package ssetest_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-sse/ssetest"
)

// TestCompatReExports exercises every sseparse re-export in ssetest at least
// once. The parser's own behavior is pinned by the sseparse module's suite;
// this test guards the delegation contract — each wrapper must pass the same
// arguments through and return the same results, so existing ssetest consumers
// keep working unchanged after the module split.
func TestCompatReExports(t *testing.T) {
	t.Parallel()

	const wire = "event: feed\n" +
		"id: 42\n" +
		"retry: 3000\n" +
		"data: {\"hello\": true}\n" +
		"data: second line\n\n" +
		"event: status\n" +
		"data: ok\n\n"

	t.Run("readers", func(t *testing.T) {
		t.Parallel()

		events := ssetest.MustReadEvents(t, strings.NewReader(wire))
		ssetest.RequireEventCount(t, events, 2)

		if _, err := ssetest.ReadEvents(strings.NewReader(wire)); err != nil {
			t.Fatalf("ReadEvents: %v", err)
		}

		if _, err := ssetest.ReadEvents(
			strings.NewReader(wire),
			ssetest.WithMaxLineBytes(64),
		); err != nil {
			t.Fatalf("ReadEvents with option: %v", err)
		}

		two, err := ssetest.ReadNEvents(strings.NewReader(wire), 2)
		if err != nil {
			t.Fatalf("ReadNEvents: %v", err)
		}

		ssetest.RequireEventCount(t, two, 2)

		mustTwo := ssetest.MustReadNEvents(t, strings.NewReader(wire), 1)
		ssetest.RequireEventCount(t, mustTwo, 1)

		reader := ssetest.NewStreamReader(strings.NewReader(wire))
		first := ssetest.MustReadNextEvent(t, reader)
		ssetest.RequireEventType(t, first, "feed")

		second, err := reader.Next()
		if err != nil {
			t.Fatalf("StreamReader.Next: %v", err)
		}

		ssetest.RequireEventType(t, second, "status")
	})

	t.Run("finders and debug output", func(t *testing.T) {
		t.Parallel()

		events := ssetest.MustReadEvents(t, strings.NewReader(wire))

		evt, found := ssetest.FindByType(events, "status")
		if !found {
			t.Fatal("FindByType should find the status event")
		}

		ssetest.RequireDataContains(t, evt, "ok")

		feeds := ssetest.FilterByType(events, "feed")
		ssetest.RequireEventCount(t, feeds, 1)

		if debug := ssetest.EventsString(events); !strings.Contains(debug, "type=feed") {
			t.Errorf("EventsString should describe events; got %q", debug)
		}
	})

	t.Run("assertions", func(t *testing.T) {
		t.Parallel()

		events := ssetest.MustReadEvents(t, strings.NewReader(wire))

		ssetest.RequireEventID(t, events[0], "42")
		ssetest.RequireRetry(t, events[0], 3000)
		ssetest.RequireDataJSON(t, ssetest.Event{
			DataLines: []string{`{"hello": true}`},
		}, map[string]any{"hello": true})
	})
}
