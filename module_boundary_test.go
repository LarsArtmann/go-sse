package sse_test

import (
	"os"
	"strings"
	"testing"
)

// TestRootModuleDoesNotRequireSsetest guards the module boundary: the root
// module must never require ssetest. ssetest is a consumer-side test-helpers
// module that reaches the root via `replace go-sse => ..` — a root require
// would create a circular module dependency. If this test fails, someone added
// ssetest to go.mod; move the test into ssetest/ instead.
func TestRootModuleDoesNotRequireSsetest(t *testing.T) {
	t.Parallel()

	goMod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	const ssetestPath = "github.com/larsartmann/go-sse/ssetest"

	for line := range strings.SplitSeq(string(goMod), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "//") {
			continue
		}

		if strings.Contains(line, ssetestPath) {
			t.Errorf("root go.mod references %q — root must never require ssetest "+
				"(circular dependency). Move the test to ssetest/ instead.",
				ssetestPath)
		}
	}
}
