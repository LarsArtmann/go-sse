//go:build ignore

// gen_corpus is the WPT-ingestion half of the corpus contract: it decodes
// testdata/wpt_format_corpus.json, optionally ingests new vectors from
// testdata/corpus_pending.json, and re-marshals everything through the
// canonical recipe the byte-form gate pins (encoding/json MarshalIndent,
// two-space indent, trailing newline). Running it with no pending file is
// the idempotence proof: the output must equal the checked-in file.
//
// Usage (from sseparse/):
//
//	go generate .
//	# or explicitly:
//	go run gen_corpus.go
//
// Ingest procedure: add fully-formed vectors (name, url, wire, AND events)
// to testdata/corpus_pending.json in the same shape as the corpus, run the
// generator, then delete the pending file. Every new vector is executed
// through the real reader before it is allowed in — a vector whose events
// do not match what sseparse dispatches aborts the run, so expectations
// cannot drift from the parser. Names are unique across the corpus; a
// pending vector whose name duplicates an existing, byte-identical vector
// is skipped, while a duplicate name with different content aborts.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/larsartmann/go-sse/sseparse"
)

const (
	corpusPath  = "testdata/wpt_format_corpus.json"
	pendingPath = "testdata/corpus_pending.json"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gen_corpus: %v\n", err)

		os.Exit(1)
	}
}

func run() error {
	raw, err := os.ReadFile(corpusPath)
	if err != nil {
		return fmt.Errorf("read corpus: %w", err)
	}

	vectors, err := decode(raw)
	if err != nil {
		return err
	}

	known := make(map[string]int, len(vectors))

	for i, v := range vectors {
		if err := validateVector(v); err != nil {
			return fmt.Errorf("corpus vector %d (%s): %w", i, v.Name, err)
		}

		known[v.Name] = i
	}

	added := 0

	if _, statErr := os.Stat(pendingPath); statErr == nil {
		vectors, added, err = ingest(vectors, known)
		if err != nil {
			return err
		}
	}

	canon, err := canonical(vectors)
	if err != nil {
		return err
	}

	if bytes.Equal(raw, canon) {
		fmt.Printf("corpus already canonical: %d vectors, %d ingested\n", len(vectors), added)

		return nil
	}

	if writeErr := os.WriteFile(corpusPath, canon, 0o644); writeErr != nil {
		return fmt.Errorf("write corpus: %w", writeErr)
	}

	fmt.Printf("corpus rewritten canonically: %d vectors, %d ingested\n", len(vectors), added)

	return nil
}

// decode decodes the corpus bytes with the same encoder the canonical gate
// uses, so field-by-field drift between generator and gate cannot happen.
func decode(raw []byte) ([]sseparse.ConformanceVector, error) {
	var vectors []sseparse.ConformanceVector

	if err := json.Unmarshal(raw, &vectors); err != nil {
		return nil, fmt.Errorf("decode corpus: %w", err)
	}

	return vectors, nil
}

// canonical renders the corpus through the pinned byte recipe.
func canonical(vectors []sseparse.ConformanceVector) ([]byte, error) {
	canon, err := json.MarshalIndent(vectors, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal corpus: %w", err)
	}

	return append(canon, '\n'), nil
}

// validateVector enforces the same contract TestCorpusIntegrity pins, so a
// vector rejected here never reaches the gates downstream.
func validateVector(v sseparse.ConformanceVector) error {
	if v.Name == "" {
		return fmt.Errorf("empty name")
	}

	if v.URL == "" {
		return fmt.Errorf("empty url (every vector must cite its upstream source)")
	}

	if v.Wire == "" {
		return fmt.Errorf("empty wire")
	}

	return nil
}

// ingest merges testdata/corpus_pending.json into vectors. Every pending
// vector is executed through the real reader; its Events field must equal
// what the parser dispatches or the run aborts. Returns the grown slice —
// the caller must take it, because append cannot grow a slice in place —
// and the ingested count.
func ingest(vectors []sseparse.ConformanceVector, known map[string]int) ([]sseparse.ConformanceVector, int, error) {
	pendingRaw, err := os.ReadFile(pendingPath)
	if err != nil {
		return vectors, 0, fmt.Errorf("read pending: %w", err)
	}

	pending, err := decode(pendingRaw)
	if err != nil {
		return vectors, 0, fmt.Errorf("decode pending: %w", err)
	}

	added := 0

	for _, v := range pending {
		if err := validateVector(v); err != nil {
			return vectors, added, fmt.Errorf("pending vector %q: %w", v.Name, err)
		}

		dispatched, err := sseparse.ReadEvents(strings.NewReader(v.Wire))
		if err != nil {
			return vectors, added, fmt.Errorf("pending vector %q (%s): wire does not parse: %w", v.Name, v.URL, err)
		}

		if err := requireDispatchMatch(v, dispatched); err != nil {
			return vectors, added, err
		}

		if idx, dup := known[v.Name]; dup {
			existingRaw, existingErr := json.Marshal(vectors[idx])
			pendingRaw, pendingErr := json.Marshal(v)

			if existingErr == nil && pendingErr == nil && bytes.Equal(existingRaw, pendingRaw) {
				continue
			}

			return vectors, added, fmt.Errorf("pending vector %q duplicates corpus vector %d with different content", v.Name, idx)
		}

		vectors = append(vectors, v)
		known[v.Name] = len(vectors) - 1
		added++
	}

	return vectors, added, nil
}

// requireDispatchMatch asserts the vector's declared Events equal what the
// real parser dispatched, using the same projection the conformance test
// asserts with (Type, joined Data, ID, Retry).
func requireDispatchMatch(v sseparse.ConformanceVector, dispatched []sseparse.Event) error {
	if len(dispatched) != len(v.Events) {
		return fmt.Errorf(
			"pending vector %q (%s): declared %d events but parser dispatches %d\nwire: %q",
			v.Name, v.URL, len(v.Events), len(dispatched), v.Wire,
		)
	}

	for i, got := range dispatched {
		want := v.Events[i]

		switch {
		case got.Type != want.Type:
			return fmt.Errorf("pending vector %q: event %d type %q, declared %q", v.Name, i, got.Type, want.Type)
		case got.Data() != want.Data:
			return fmt.Errorf("pending vector %q: event %d data %q, declared %q", v.Name, i, got.Data(), want.Data)
		case got.ID != want.ID:
			return fmt.Errorf("pending vector %q: event %d id %q, declared %q", v.Name, i, got.ID, want.ID)
		case got.Retry != want.Retry:
			return fmt.Errorf("pending vector %q: event %d retry %d, declared %d", v.Name, i, got.Retry, want.Retry)
		}
	}

	return nil
}
