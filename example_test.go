package sse_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/larsartmann/go-sse"
)

// ExampleWriteEvent demonstrates the SSE wire format produced by WriteEvent.
// Each field maps to an SSE line; the frame ends with a blank line.
func ExampleWriteEvent() {
	var buf bytes.Buffer

	_ = sse.WriteEvent(&buf, sse.Event{
		Event: "update",
		Data:  "hello",
		ID:    sse.NewEventID("42"),
	})

	fmt.Print(buf.String())

	// Output:
	// event: update
	// data: hello
	// id: 42
}

// ExampleBroadcaster demonstrates fan-out: one Broadcast reaches every
// subscriber. Broadcast is non-blocking; slow consumers miss events when
// their 64-deep buffer is full.
func ExampleBroadcaster() {
	bc := sse.NewBroadcaster[sse.Event]()

	ch := bc.Subscribe()
	defer bc.Unsubscribe(ch)

	bc.Broadcast(sse.Event{Event: "msg", Data: "hi"})

	evt := <-ch
	fmt.Println(evt)

	// Output: {event:msg data:hi}
}

// ExampleParseEventID shows constructing an EventID from untrusted input.
// Values containing newlines or carriage returns are rejected (they would
// corrupt the SSE wire format); empty values are allowed.
func ExampleParseEventID() {
	id, err := sse.ParseEventID("evt-99")
	if err != nil {
		fmt.Println("invalid:", err)

		return
	}

	fmt.Println(id.Get())

	// Output: evt-99
}

// ExampleKeyedLines demonstrates the keyed-data-line pattern used by DataStar
// and similar SSE protocols. Each line of a multi-line value is prefixed with
// the key, producing the wire format DataStar expects.
func ExampleKeyedLines() {
	var buf bytes.Buffer

	html := "<div id=\"feed\">\n  <span>1</span>\n</div>"

	_ = sse.WriteEvent(&buf, sse.Event{
		Event: "datastar-patch-elements",
		Data: "selector #feed\nmode inner\n" +
			sse.KeyedLines("elements", html),
	})

	fmt.Print(buf.String())

	// Output:
	// event: datastar-patch-elements
	// data: selector #feed
	// data: mode inner
	// data: elements <div id="feed">
	// data: elements   <span>1</span>
	// data: elements </div>
}

// ExampleBroadcaster_SubscribeFilter demonstrates predicate-based filtering:
// only events matching the predicate are delivered to the subscriber's channel.
// Subscribe() with no filter is equivalent to SubscribeFilter(nil).
func ExampleBroadcaster_SubscribeFilter() {
	bc := sse.NewBroadcaster[sse.Event]()

	// Only receive "message" events, skip everything else
	msgs := bc.SubscribeFilter(func(evt sse.Event) bool {
		return evt.Event == "message"
	})
	defer bc.Unsubscribe(msgs)

	bc.Broadcast(sse.Event{Event: "message", Data: "hello"})
	bc.Broadcast(sse.Event{Event: "reaction", Data: "ignored"})
	bc.Broadcast(sse.Event{Event: "message", Data: "world"})

	fmt.Println(<-msgs)
	fmt.Println(<-msgs)

	// Output:
	// {event:message data:hello}
	// {event:message data:world}
}

// ExampleReplayFiltered demonstrates predicate-based reconnection replay:
// after a client reconnects (empty Last-Event-ID means "from the beginning"),
// only events matching the predicate are sent. Here a minimal in-memory store
// is filtered to "message" events; the "reaction" event is skipped.
//
// The store here is a test helper; in real code you back this with a database
// or ring buffer that implements sse.EventStore.
func ExampleReplayFiltered() {
	store := &memoryStore{events: []sse.Event{
		{Event: "message", Data: "hello", ID: sse.NewEventID("1")},
		{Event: "reaction", Data: "wave", ID: sse.NewEventID("2")},
		{Event: "message", Data: "world", ID: sse.NewEventID("3")},
	}}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	n, _ := sse.ReplayFiltered(stream, store, sse.NewEventID(""),
		func(evt sse.Event) bool { return evt.Event == "message" })

	fmt.Println("replayed", n, "events")
	fmt.Print(w.Body.String())

	// Output:
	// replayed 2 events
	// event: message
	// data: hello
	// id: 1
	//
	// event: message
	// data: world
	// id: 3
}
