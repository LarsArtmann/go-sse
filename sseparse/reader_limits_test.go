package sseparse_test

import (
	"bufio"
	"errors"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse/sseparse"
)

// lineOfLen returns a data line ("data:" + fill) of exactly totalLen bytes,
// terminated by "\n\n" so it dispatches one event.
func lineOfLen(totalLen int) string {
	const prefixLen = len("data:")

	return "data:" + strings.Repeat("x", max(totalLen-prefixLen, 0)) + "\n\n"
}

// TestReadEvents_LineCapDefault pins the 1 MiB per-line boundary: a line one
// byte under [sseparse.DefaultMaxLineBytes] parses (its terminator is the
// last byte that fits the cap), while a line exactly at the cap fails the
// scan with bufio.ErrTooLong wrapped in the returned error.
func TestReadEvents_LineCapDefault(t *testing.T) {
	t.Parallel()

	t.Run("line one byte under cap parses", func(t *testing.T) {
		t.Parallel()

		events, err := sseparse.ReadEvents(
			strings.NewReader(lineOfLen(sseparse.DefaultMaxLineBytes - 1)),
		)
		if err != nil {
			t.Fatalf("line under cap should parse: %v", err)
		}

		sseparse.RequireEventCount(t, events, 1)
	})

	t.Run("line at cap fails", func(t *testing.T) {
		t.Parallel()

		events, err := sseparse.ReadEvents(
			strings.NewReader(lineOfLen(sseparse.DefaultMaxLineBytes)),
		)
		if err == nil {
			t.Fatal("line at cap should fail the scan (terminator no longer fits)")
		}

		if len(events) != 0 {
			t.Errorf("over-cap scan should return 0 events; got %d", len(events))
		}

		if !errors.Is(err, bufio.ErrTooLong) {
			t.Errorf("over-cap error should wrap bufio.ErrTooLong: got %v", err)
		}
	})
}

// TestReadEvents_WithMaxLineBytes pins the configurable line cap: a lowered
// cap rejects lines the default would accept, a raised cap accepts lines the
// default would reject, and a non-positive value is ignored (default kept).
func TestReadEvents_WithMaxLineBytes(t *testing.T) {
	t.Parallel()

	t.Run("lowered cap rejects", func(t *testing.T) {
		t.Parallel()

		_, err := sseparse.ReadEvents(
			strings.NewReader(lineOfLen(64)),
			sseparse.WithMaxLineBytes(32),
		)
		if !errors.Is(err, bufio.ErrTooLong) {
			t.Errorf("64-byte line under 32-byte cap: got %v, want bufio.ErrTooLong", err)
		}
	})

	t.Run("raised cap accepts", func(t *testing.T) {
		t.Parallel()

		events, err := sseparse.ReadEvents(
			strings.NewReader(lineOfLen(128)),
			sseparse.WithMaxLineBytes(256),
		)
		if err != nil {
			t.Fatalf("128-byte line under 256-byte cap should parse: %v", err)
		}

		sseparse.RequireEventCount(t, events, 1)
	})

	t.Run("non-positive keeps default", func(t *testing.T) {
		t.Parallel()

		events, err := sseparse.ReadEvents(
			strings.NewReader(lineOfLen(64)),
			sseparse.WithMaxLineBytes(0),
		)
		if err != nil {
			t.Fatalf("64-byte line with non-positive option (default kept) should parse: %v", err)
		}

		sseparse.RequireEventCount(t, events, 1)
	})
}

// TestReadNEvents_WithMaxLineBytes and TestStreamReader_WithMaxLineBytes pin
// that every entry point honors the same ReadOption plumbing.
func TestReadNEvents_WithMaxLineBytes(t *testing.T) {
	t.Parallel()

	_, err := sseparse.ReadNEvents(
		strings.NewReader(lineOfLen(64)),
		1,
		sseparse.WithMaxLineBytes(32),
	)
	if !errors.Is(err, bufio.ErrTooLong) {
		t.Errorf("ReadNEvents over cap: got %v, want bufio.ErrTooLong", err)
	}
}

func TestStreamReader_WithMaxLineBytes(t *testing.T) {
	t.Parallel()

	reader := sseparse.NewStreamReader(
		strings.NewReader(lineOfLen(64)),
		sseparse.WithMaxLineBytes(32),
	)

	_, err := reader.Next()
	if !errors.Is(err, bufio.ErrTooLong) {
		t.Errorf("StreamReader over cap: got %v, want bufio.ErrTooLong", err)
	}
}
