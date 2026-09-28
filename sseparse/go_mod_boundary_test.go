package sseparse_test

import (
	"os"
	"strings"
	"testing"
)

// TestModuleStaysZeroDependency guards sseparse's reason to exist as its own
// module: NO third-party requires. Consumers adopt sseparse precisely so
// go-sse, go-branded-id, and go-error-family stay out of their module graph
// (private-repo hermetic builds re-hash every module in the graph); ssetest
// re-exports this package for consumers who DO want the HTTP helpers. If this
// test fails, someone added a require — either vendor the logic inline, move
// it behind an interface, or relocate it to ssetest.
func TestModuleStaysZeroDependency(t *testing.T) {
	t.Parallel()

	goMod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	for line := range strings.SplitSeq(string(goMod), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "module ") {
			continue
		}

		if strings.Contains(line, "=>") { // replace directives fail separately
			t.Errorf("zero-dependency module must not carry replace directives: %q", line)
		}

		if !strings.Contains(line, "github.com/") && !strings.Contains(line, "golang.org/") {
			continue
		}

		t.Errorf("go.mod requires a third-party module (%q) — sseparse must stay "+
			"dependency-free; see the test comment for alternatives", line)
	}
}
