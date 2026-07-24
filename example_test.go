package sse_test

import (
	"bytes"
	"fmt"

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
