package ssetest_test

import (
	"fmt"
	"strings"
	"testing"

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

// ExampleRequireDataJSON asserts a JSON payload on a collected event inside a
// handler test: collect the handler's events, then require that the first
// one's data unmarshals into (and equals) the expected value. The tb argument
// is the test's *testing.T; every helper takes testing.TB so benchmarks and
// BDD suites (e.g. Ginkgo's GinkgoT()) work unchanged.
//
// This example is compile-only (it renders on pkg.go.dev; there is no live
// *testing.T inside an example function, and it declares no Output).
func ExampleRequireDataJSON() {
	var tb testing.TB

	events := ssetest.MustReadEvents(tb, strings.NewReader(
		"event: status\ndata: {\"done\": true}\n\n"))

	ssetest.RequireDataJSON(tb, events[0], map[string]any{"done": true})
}
