package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
)

func makeTestEvent(id int64) sse.Event {
	return sse.Event{
		Event: eventPatchElements,
		Data:  "selector #feed\nmode prepend\ncategory info\nelements <div>test</div>",
		ID:    sse.NewEventID(strconv.FormatInt(id, 10)),
	}
}

// collectSSEEvents connects to an SSE endpoint, reads events for the given
// duration, and returns the raw event blocks as strings.
func collectSSEEvents(url string, duration time.Duration) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), duration+2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connect to SSE endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return scanSSEFrames(resp, duration)
}

// scanSSEFrames reads SSE wire frames from resp until duration elapses or the
// stream ends. Blank lines separate frames; each frame is returned with its
// lines joined by newlines.
func scanSSEFrames(resp *http.Response, duration time.Duration) ([]string, error) {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Read lines on a goroutine so the timer fires even when no new
	// data is arriving on the stream (the old busy-wait pattern
	// blocked indefinitely inside scanner.Scan between events).
	type scanLine struct {
		text string
		ok   bool
	}

	lineCh := make(chan scanLine)

	go func() {
		for scanner.Scan() {
			lineCh <- scanLine{text: scanner.Text(), ok: true}
		}
		lineCh <- scanLine{ok: false}
	}()

	timer := time.NewTimer(duration)
	defer timer.Stop()

	var (
		events  []string
		current strings.Builder
	)

	for {
		select {
		case <-timer.C:
			return flushFrame(events, &current), nil
		case line := <-lineCh:
			if !line.ok {
				return finishScan(events, &current, scanner)
			}
			if line.text == "" {
				events = flushFrame(events, &current)
			} else {
				current.WriteString(line.text)
				current.WriteString("\n")
			}
		}
	}
}

// flushFrame appends the pending frame (if any) and resets the builder.
func flushFrame(events []string, current *strings.Builder) []string {
	if current.Len() == 0 {
		return events
	}

	events = append(events, current.String())
	current.Reset()

	return events
}

// finishScan finalizes the last pending frame when the stream ends and maps
// scanner errors, tolerating the context cancellation that stops collection.
func finishScan(
	events []string,
	current *strings.Builder,
	scanner *bufio.Scanner,
) ([]string, error) {
	events = flushFrame(events, current)

	scanErr := scanner.Err()
	if scanErr != nil &&
		!errors.Is(scanErr, context.Canceled) &&
		!errors.Is(scanErr, context.DeadlineExceeded) {
		return events, fmt.Errorf("scan SSE stream: %w", scanErr)
	}

	return events, nil
}

// extractEventIDs pulls the id: field values from SSE event blocks.
func extractEventIDs(events []string) []string {
	var ids []string

	for _, evt := range events {
		for line := range strings.SplitSeq(evt, "\n") {
			if rest, ok := strings.CutPrefix(line, "id:"); ok {
				id := strings.TrimSpace(rest)
				if id != "" {
					ids = append(ids, id)
				}
			}
		}
	}

	return ids
}

// commonIDs returns IDs present in both slices.
func commonIDs(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, id := range b {
		set[id] = true
	}

	var result []string

	for _, id := range a {
		if set[id] {
			result = append(result, id)
		}
	}

	return result
}

// connectSSEClient opens an SSE connection and returns a cancel function
// to close it. Used for subscriber-count and shutdown tests.
func connectSSEClient(t *testing.T, url string) context.CancelFunc {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	req.Header.Set("Accept", "text/event-stream")

	var wg sync.WaitGroup

	wg.Go(func() {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}

		defer func() { _ = resp.Body.Close() }()

		_, _ = io.Copy(io.Discard, resp.Body)
	})

	return func() {
		cancel()
		wg.Wait()
	}
}
