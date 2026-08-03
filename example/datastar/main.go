// Package main implements a DataStar-compatible SSE server using go-sse.
//
// Demonstrates both DataStar event types:
//   - datastar-patch-signals: updates client-side reactive state ($progress)
//   - datastar-patch-elements: patches DOM elements (#status)
//
// Run: go run example/datastar/main.go
// Then: open http://localhost:8081 in your browser
package main

import (
	"encoding/json/v2"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/larsartmann/go-sse"
)

const (
	datastarAddr  = ":8765"
	progressStep  = 10
	maxProgress   = 100
	progressDelay = 500 * time.Millisecond
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", indexHandler)
	mux.HandleFunc("GET /events", eventsHandler)

	log.Printf("DataStar example on http://localhost%s", datastarAddr)
	log.Print("Open the URL in your browser to see live SSE-driven DOM patches.")
	log.Fatal(http.ListenAndServe( //nolint:gosec // G114: intentional for example server
		datastarAddr,
		mux,
	))
}

func indexHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, indexHTML)
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	ctx := stream.Context()

	for progress := 0; progress <= maxProgress; progress += progressStep {
		select {
		case <-ctx.Done():
			return
		default:
		}

		sendProgress(stream, progress)

		if progress < maxProgress {
			select {
			case <-ctx.Done():
				return
			case <-time.After(progressDelay):
			}
		}
	}

	_ = stream.SendLines(
		"datastar-patch-elements",
		"selector #status",
		"mode inner",
		sse.KeyedLines("elements", "<p>Complete!</p>"),
	)
}

func sendProgress(stream *sse.Stream, progress int) {
	signals, err := json.Marshal(map[string]any{"progress": progress})
	if err != nil {
		log.Printf("marshal signals: %v", err)

		return
	}

	_ = stream.SendKeyed("datastar-patch-signals", "signals", string(signals))

	status := fmt.Sprintf("<p>Processing... %d%%</p>", progress)

	_ = stream.SendLines(
		"datastar-patch-elements",
		"selector #status",
		"mode inner",
		sse.KeyedLines("elements", status),
	)
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>go-sse + DataStar</title>
  <script type="module" src="https://cdn.jsdelivr.net/gh/starfederation/[email protected]/bundles/datastar.js"></script>
  <style>
    body {
      font-family: system-ui, sans-serif;
      max-width: 640px;
      margin: 3rem auto;
      padding: 0 1.5rem;
      color: #1a1a1a;
    }
    code { background: #f0f0f0; padding: 0.15em 0.35em; border-radius: 3px; font-size: 0.9em; }
    .bar {
      background: #e0e0e0;
      border-radius: 6px;
      overflow: hidden;
      height: 28px;
      margin: 1rem 0;
    }
    .fill {
      background: linear-gradient(90deg, #4a90d9, #5cb85c);
      height: 100%;
      transition: width 0.3s ease;
    }
    button {
      padding: 0.5rem 1rem;
      font-size: 1rem;
      cursor: pointer;
      border: 1px solid #ccc;
      border-radius: 4px;
      background: #fff;
    }
    button:hover { background: #f5f5f5; }
  </style>
</head>
<body>
  <div data-signals="{progress: 0}">
    <h1>go-sse &#215; DataStar</h1>

    <p>This page receives <strong>Server-Sent Events</strong> from a Go backend built with
      <code>github.com/larsartmann/go-sse</code> and renders them with
      <a href="https://data-star.dev">DataStar</a>. No frontend JavaScript required.</p>

    <button data-on:click="@get('/events')">Restart demo</button>

    <div data-init="@get('/events')"></div>

    <div class="bar">
      <div class="fill" data-style:width="$progress + '%'"></div>
    </div>

    <div id="status"><p>Connecting&#8230;</p></div>

    <p>Signal value: <span data-text="$progress"></span>%</p>
  </div>
</body>
</html>
`
