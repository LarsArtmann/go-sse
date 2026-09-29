package sseparse

import (
	"fmt"
	"strings"
	"testing"
)

// fatalTB captures Fatalf calls so MustCorpus's failure branch is testable
// without aborting the test binary. Same pattern as the recordingTB in
// assert_test.go, which lives in the external test package and therefore
// cannot reach the package-internal corpusJSON var this needs to corrupt.
type fatalTB struct {
	testing.TB

	fatals []string
}

func (tb *fatalTB) Helper() {}

func (tb *fatalTB) Fatalf(format string, args ...any) {
	tb.fatals = append(tb.fatals, fmt.Sprintf(format, args...))
}

// corruptCorpusForTest swaps the embedded corpus bytes for garbage and
// restores them on cleanup. Deliberately not parallel: corpusJSON is package
// state, and the parked parallel conformance tests must not observe the
// mutation (sequential tests finish before parallel ones resume).
func corruptCorpusForTest(t *testing.T) {
	t.Helper()

	original := corpusJSON
	corpusJSON = []byte("{not json")
	t.Cleanup(func() { corpusJSON = original })
}

func TestCorpus_DecodeFailure(
	t *testing.T,
) { //nolint:paralleltest // mutates corpusJSON; parallel corpus readers must not observe it
	corruptCorpusForTest(t)

	vectors, err := Corpus()
	if err == nil {
		t.Fatalf("Corpus() succeeded on corrupt JSON: %d vectors", len(vectors))
	}

	if !strings.Contains(err.Error(), "decode embedded WPT format corpus") {
		t.Errorf("error should name the corpus decode, got: %v", err)
	}
}

func TestMustCorpus_FatalsOnCorruptJSON(
	t *testing.T,
) { //nolint:paralleltest // mutates corpusJSON; parallel corpus readers must not observe it
	corruptCorpusForTest(t)

	tb := &fatalTB{}
	vectors := MustCorpus(tb)

	if len(tb.fatals) != 1 {
		t.Fatalf("MustCorpus recorded %d fatals, want 1: %v", len(tb.fatals), tb.fatals)
	}

	if !strings.Contains(tb.fatals[0], "load WPT format corpus") {
		t.Errorf("fatal should name the corpus load, got: %q", tb.fatals[0])
	}

	if vectors != nil {
		t.Errorf("MustCorpus returned %d vectors after a fatal, want nil", len(vectors))
	}
}
