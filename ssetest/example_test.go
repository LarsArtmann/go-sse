package ssetest_test

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-sse/ssetest"
)

// ExampleReadEvents demonstrates parsing raw SSE wire output into assertable
// events — useful when a test already captured the response body elsewhere.
func ExampleReadEvents() {
	wire := "event: feed\n" +
		"id: 42\n" +
		"data: first line\n" +
		"data: second line\n\n"

	events, _ := ssetest.ReadEvents(strings.NewReader(wire))

	fmt.Println("type:", events[0].Type)
	fmt.Println("id:", events[0].ID)
	fmt.Println("data:", events[0].Data())
	fmt.Println("datalines:", len(events[0].DataLines))
	// Output:
	// type: feed
	// id: 42
	// data: first line
	// second line
	// datalines: 2
}

// ExampleEventsString shows the debug representation for test failure
// messages.
func ExampleEventsString() {
	wire := "event: feed\ndata: hello\n\n"
	events, _ := ssetest.ReadEvents(strings.NewReader(wire))

	fmt.Println(ssetest.EventsString(events))
	// Output:
	// Event{type=feed datalines=1}
}
