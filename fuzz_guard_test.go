package sse_test

import (
	"os"
	"strings"
	"testing"
)

// TestFuzzTargetsLiveHere guards the CI fuzz job's module mapping: every
// target the workflow runs in THIS module must exist in this module's test
// files. When a fuzz function moves to another module, `-fuzz=Name` in the
// old module exits 0 with only a stderr warning ("no fuzz tests to fuzz") —
// a green no-op that silently shrinks the fuzz budget. This test turns that
// silence red. The three modules each carry their own copy of this guard:
// module isolation forbids sharing a helper across them.
func TestFuzzTargetsLiveHere(t *testing.T) {
	t.Parallel()

	assertFuzzTargets(t, []string{"FuzzWriteEvent", "FuzzParseEventID", "FuzzKeyedLines"})
}

// assertFuzzTargets fails when any expected `func Fuzz<Name>(f *testing.F)`
// is missing from the package's test files.
func assertFuzzTargets(t *testing.T, want []string) {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("list package dir: %v", err)
	}

	var testFiles []string

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.go") {
			testFiles = append(testFiles, entry.Name())
		}
	}

	for _, name := range want {
		signature := "func " + name + "(f *testing.F)"
		found := false

		for _, file := range testFiles {
			content, readErr := os.ReadFile(file)
			if readErr != nil {
				t.Fatalf("read %s: %v", file, readErr)
			}

			if strings.Contains(string(content), signature) {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("fuzz target %s is gone from this module — CI runs `-fuzz=%s` "+
				"here, so it would silently fuzz nothing; move the ci.yml step with "+
				"the target or restore it", name, name)
		}
	}
}
