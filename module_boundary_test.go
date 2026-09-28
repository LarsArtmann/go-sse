package sse_test

import (
	"os"
	"strings"
	"testing"
)

// TestRootModuleDoesNotRequireSubmodules guards the module boundaries: the
// root module must never require ssetest or sseparse. Both are consumer-side
// modules that reach the root via `replace go-sse => ..` — a root require
// would create a circular module dependency. If this test fails, someone
// added a submodule path to go.mod; move the test into that submodule instead.
func TestRootModuleDoesNotRequireSubmodules(t *testing.T) {
	t.Parallel()

	goMod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	const (
		ssetestPath  = "github.com/larsartmann/go-sse/ssetest"
		sseparsePath = "github.com/larsartmann/go-sse/sseparse"
	)

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

		if strings.Contains(line, sseparsePath) {
			t.Errorf("root go.mod references %q — root must never require sseparse "+
				"(circular dependency). Move the test to sseparse/ instead.",
				sseparsePath)
		}
	}
}
