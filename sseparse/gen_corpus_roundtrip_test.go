package sseparse_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse/sseparse"
)

// testFilePerm names the file mode for sandbox fixtures (keeps magic-number
// linters quiet and the intent local).
const testFilePerm = 0o600

// TestGenCorpusRoundTrip exercises the checked-in ingestion generator as a
// process — the exact command `go generate` runs (`go run gen_corpus.go`) —
// so the ingestion proof lives in CI instead of a session transcript. It
// pins the five properties of the corpus contract on a throwaway sandbox per
// scenario, never touching the real testdata corpus:
//
//  1. idempotence: a canonical corpus re-marshals to identical bytes,
//  2. genuine ingestion: a new dispatch-valid vector is appended canonically,
//  3. duplicate skip: a byte-identical pending vector changes nothing,
//  4. duplicate drift abort: same name, different content fails the run,
//  5. dispatch mismatch abort: events that disagree with the real parser
//     fail the run.
//
// A pending vector is only allowed in when sseparse actually dispatches what
// its Events field declares — properties 4 and 5 are the teeth of that rule.
func TestGenCorpusRoundTrip(t *testing.T) {
	t.Parallel()

	original, err := os.ReadFile("testdata/wpt_format_corpus.json")
	if err != nil {
		t.Fatalf("read checked-in corpus: %v", err)
	}

	corpus, err := sseparse.Corpus()
	if err != nil {
		t.Fatalf("decode checked-in corpus: %v", err)
	}

	if len(corpus) == 0 {
		t.Fatal("checked-in corpus is empty — cannot build round-trip scenarios")
	}

	base := corpus[0]

	t.Run("idempotent on canonical corpus", func(t *testing.T) {
		t.Parallel()

		dir := newGenSandbox(t, original)
		stdout, exitErr := runGenerator(t, dir)
		if exitErr != nil {
			t.Fatalf("generator failed on a canonical corpus: %v\n%s", exitErr, stdout)
		}

		if !strings.Contains(stdout, "already canonical") {
			t.Errorf("expected idempotence report, got: %s", stdout)
		}

		assertSandboxCorpusUnchanged(t, dir, original)
	})

	t.Run("ingests a new dispatch-valid vector", func(t *testing.T) {
		t.Parallel()

		dir := newGenSandbox(t, original)

		ingested := base
		ingested.Name = "gen-roundtrip-ingested"
		writePending(t, dir, []sseparse.ConformanceVector{ingested})

		stdout, exitErr := runGenerator(t, dir)
		if exitErr != nil {
			t.Fatalf("generator rejected a dispatch-valid vector: %v\n%s", exitErr, stdout)
		}

		if !strings.Contains(stdout, "1 ingested") {
			t.Errorf("expected 1 ingested vector, got: %s", stdout)
		}

		grown := readSandboxCorpus(t, dir)
		if len(grown) != len(corpus)+1 {
			t.Fatalf("corpus has %d vectors after ingestion, want %d", len(grown), len(corpus)+1)
		}
	})

	t.Run("skips a byte-identical duplicate", func(t *testing.T) {
		t.Parallel()

		dir := newGenSandbox(t, original)
		writePending(t, dir, []sseparse.ConformanceVector{base})

		stdout, exitErr := runGenerator(t, dir)
		if exitErr != nil {
			t.Fatalf("generator aborted on a byte-identical duplicate: %v\n%s", exitErr, stdout)
		}

		if !strings.Contains(stdout, "0 ingested") {
			t.Errorf("expected duplicate to be skipped, got: %s", stdout)
		}

		assertSandboxCorpusUnchanged(t, dir, original)
	})

	t.Run("aborts on duplicate with drifted content", func(t *testing.T) {
		t.Parallel()

		dir := newGenSandbox(t, original)

		drifted := base
		drifted.Notes = base.Notes + " (content drifted by the round-trip test)"
		writePending(t, dir, []sseparse.ConformanceVector{drifted})

		if stdout, exitErr := runGenerator(t, dir); exitErr == nil {
			t.Fatalf("generator accepted a drifted duplicate:\n%s", stdout)
		}

		assertSandboxCorpusUnchanged(t, dir, original)
	})

	t.Run("aborts when declared events disagree with the parser", func(t *testing.T) {
		t.Parallel()

		dir := newGenSandbox(t, original)

		mismatched := base
		mismatched.Name = "gen-roundtrip-mismatch"
		if len(mismatched.Events) == 0 {
			t.Skip("base vector declares no events; nothing to mismatch")
		}

		mismatched.Events[0].Data += "\n(round-trip drift)"

		writePending(t, dir, []sseparse.ConformanceVector{mismatched})

		if stdout, exitErr := runGenerator(t, dir); exitErr == nil {
			t.Fatalf("generator accepted a vector whose events disagree with the parser:\n%s", stdout)
		}

		assertSandboxCorpusUnchanged(t, dir, original)
	})
}

// newGenSandbox materializes a throwaway generator workspace: the generator
// source and a canonical corpus copy, so scenarios can mutate freely without
// touching testdata.
func newGenSandbox(t *testing.T, corpus []byte) string {
	t.Helper()

	dir := t.TempDir()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("list package dir: %v", err)
	}

	// Mirror the real layout: the generator (package main, build-ignored)
	// sits beside the package sources it imports, so the sandbox needs every
	// non-test .go file plus the module definition.
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		gen, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		if err := os.WriteFile(filepath.Join(dir, name), gen, testFilePerm); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	goMod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), goMod, testFilePerm); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(dir, "testdata"), 0o750); err != nil {
		t.Fatalf("create sandbox testdata: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "testdata", "wpt_format_corpus.json"), corpus, testFilePerm); err != nil {
		t.Fatalf("write corpus fixture: %v", err)
	}

	return dir
}

// writePending drops a corpus_pending.json into the sandbox (the generator's
// ingestion input).
func writePending(t *testing.T, dir string, vectors []sseparse.ConformanceVector) {
	t.Helper()

	raw, err := json.Marshal(vectors)
	if err != nil {
		t.Fatalf("marshal pending vectors: %v", err)
	}

	path := filepath.Join(dir, "testdata", "corpus_pending.json")
	if err := os.WriteFile(path, raw, testFilePerm); err != nil {
		t.Fatalf("write pending vectors: %v", err)
	}
}

// runGenerator runs the checked-in generator inside the sandbox and returns
// its stdout plus the process error (non-nil iff the run exited nonzero).
// GOFLAGS/GOPROXY are pinned because the generator must work offline against
// the module itself — exactly the conditions of the hermetic nix check.
func runGenerator(t *testing.T, dir string) (string, error) {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "go", "run", "gen_corpus.go")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOPROXY=off")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		// The generator reports failures on stderr; carry it in the error so
		// scenario failures are diagnosable without rerunning by hand.
		return stdout.String(), fmt.Errorf("%w\nstderr: %s", err, strings.TrimSpace(stderr.String()))
	}

	return stdout.String(), nil
}

// readSandboxCorpus decodes the sandbox corpus so scenarios can assert on
// the ingested vector count.
func readSandboxCorpus(t *testing.T, dir string) []sseparse.ConformanceVector {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(dir, "testdata", "wpt_format_corpus.json"))
	if err != nil {
		t.Fatalf("read sandbox corpus: %v", err)
	}

	var vectors []sseparse.ConformanceVector
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatalf("decode sandbox corpus: %v", err)
	}

	return vectors
}

// assertSandboxCorpusUnchanged fails when the sandbox corpus no longer holds
// the exact original bytes (the abort scenarios' core assertion: a rejected
// run must leave the corpus untouched).
func assertSandboxCorpusUnchanged(t *testing.T, dir string, original []byte) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(dir, "testdata", "wpt_format_corpus.json"))
	if err != nil {
		t.Fatalf("read sandbox corpus: %v", err)
	}

	if !bytes.Equal(raw, original) {
		t.Error("sandbox corpus was modified by a run that should have left it untouched")
	}
}
