// Package ssetestcore is the zero-dependency core of
// [github.com/larsartmann/go-sse/ssetest]: the WHATWG-conformant SSE wire-format
// parser, the assertion helpers that operate on parsed events, and the WPT
// conformance corpus as consumable data.
//
// It exists as its own Go module with no third-party requires, so parsers and
// non-HTTP consumers (unix-socket daemons, protocol libraries, spec oracles)
// can adopt it without pulling go-sse, go-branded-id, or go-error-family into
// their module graph. The HTTP collection helpers ([ssetest.Collect] and
// friends) live one module up in ssetest, which re-exports this package's API
// for backward compatibility.
//
// The package depends on `testing` (the Must* and Require* helpers), so like
// ssetest it is for test code and never leaks into production builds.
//
// # Quick start
//
//	events, err := ssetestcore.ReadEvents(r)
//	if err != nil {
//	    t.Fatal(err)
//	}
//	ssetestcore.RequireData(t, events[0], "hello")
//
// # Spec oracle for another parser
//
// The transcribed WPT format corpus is exported as data via [Corpus]: run your
// own parser over each vector's Wire and compare the dispatched events to the
// vector's Events — the exact vectors that pin this package's own parser, with
// no ssetestcore parsing code in the loop:
//
//	for _, vector := range ssetestcore.MustCorpus(t) {
//	    got := myParser(vector.Wire)
//	    requireEqual(t, vector.Events, got)
//	}
//
// # Ginkgo and benchmarks
//
// All helpers that take a `tb` accept [testing.TB], not *testing.T, so they
// work with *testing.T, *testing.B, and Ginkgo's GinkgoT().
package ssetestcore
