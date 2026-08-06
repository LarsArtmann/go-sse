// Package main implements a DataStar-compatible SSE server using go-sse.
//
// Demonstrates both DataStar event types:
//   - datastar-patch-signals: updates client-side reactive state ($progress)
//   - datastar-patch-elements: patches DOM elements (#status)
//
// The HTML is rendered with templ (type-safe Go templates), CSS is served from
// a real .css file, and the DataStar JS bundle is embedded — no CDN required.
//
// Run: go run example/datastar/main.go
// Then: open http://localhost:8765 in your browser
package main

import (
	"embed"
	"encoding/json/v2"
	"fmt"
	"io/fs"
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

//go:embed all:static
var staticFiles embed.FS

func main() {
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("static sub FS: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", indexHandler)
	mux.HandleFunc("GET /events", eventsHandler)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	log.Printf("DataStar example on http://localhost%s", datastarAddr)
	log.Print("Open the URL in your browser to see live SSE-driven DOM patches.")
	log.Fatal(http.ListenAndServe( //nolint:gosec // G114: intentional for example server
		datastarAddr,
		mux,
	))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := indexPage().Render(r.Context(), w); err != nil {
		log.Printf("render index page: %v", err)
	}
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
