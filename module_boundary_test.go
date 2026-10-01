package sse_test

import (
	"os"
	"strings"
	"testing"
)

// TestRootModuleDoesNotRequireSubmodules guards the module boundaries: the
// root module must never reference ssetest or sseparse — neither as a direct
// require (go.mod) nor as an indirect entry in the module graph (go.sum).
// Both nested modules are consumer-side layers that reach the root through
// tagged module requires; a root-side reference would create a circular
// module dependency. If this test fails, someone added a submodule path to
// the root module graph; move the test into that submodule instead.
func TestRootModuleDoesNotRequireSubmodules(t *testing.T) {
	t.Parallel()

	const (
		ssetestPath  = "github.com/larsartmann/go-sse/ssetest"
		sseparsePath = "github.com/larsartmann/go-sse/sseparse"
	)

	for _, file := range []string{"go.mod", "go.sum"} {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}

		for line := range strings.SplitSeq(string(content), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "//") {
				continue
			}

			if strings.Contains(line, ssetestPath) {
				t.Errorf("%s references %q — root must never depend on ssetest "+
					"(circular module dependency). Move the test to ssetest/ instead.", file, ssetestPath)
			}

			if strings.Contains(line, sseparsePath) {
				t.Errorf("%s references %q — root must never depend on sseparse "+
					"(circular module dependency). Move the test to sseparse/ instead.", file, sseparsePath)
			}
		}
	}
}
