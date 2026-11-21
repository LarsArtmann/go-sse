package sseparse_test

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestCorpusUsesSingleJSONEncoder bans encoding/json v1 from the whole
// sseparse package. The corpus recipe used to be split-brained: the generator
// marshalled with v1 while the byte-form gate re-marshalled with v2, and the
// two disagree on indentation options, HTML escaping, and omitempty-for-zero
// on numeric fields, so CI flipped TestCorpusJSONIsCanonical between green
// and a diff with no data change.
func TestCorpusUsesSingleJSONEncoder(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}

		if importedPaths(t, name)["encoding/json"] {
			t.Errorf("%s imports encoding/json (v1); the corpus path is pinned to "+
				"encoding/json/v2", name)
		}
	}
}

// TestCorpusGeneratorAndGateShareEncoder pins the two halves of the contract
// to the same encoder explicitly. The blanket v1 ban above would still pass if
// both files moved to a third encoder; this names the recipe they must share.
func TestCorpusGeneratorAndGateShareEncoder(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"gen_corpus.go", "wpt_format_corpus_test.go"} {
		imports := importedPaths(t, name)

		for _, want := range []string{"encoding/json/v2", "encoding/json/jsontext"} {
			if !imports[want] {
				t.Errorf("%s must import %q so the generator and the byte-form gate "+
					"marshal through one recipe", name, want)
			}
		}
	}
}

func importedPaths(tb testing.TB, name string) map[string]bool {
	tb.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
	if err != nil {
		tb.Fatalf("parse %s: %v", name, err)
	}

	paths := make(map[string]bool, len(file.Imports))
	for _, imp := range file.Imports {
		paths[strings.Trim(imp.Path.Value, `"`)] = true
	}

	return paths
}
