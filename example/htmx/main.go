// Package main implements an HTMX + SSE server using go-sse.
//
// Demonstrates the HTMX model: the server streams HTML fragments as named SSE
// events, and HTMX's SSE extension swaps each fragment into a target element.
// Contrast this with the DataStar example (../datastar), which patches reactive
// signals and DOM elements via a custom keyed-line protocol.
//
// Run: go run example/htmx/
// Then: open http://localhost:8766 in your browser
package main

import (
	"bytes"
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/larsartmann/go-sse"
)

const (
	htmxAddr      = ":8766"
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
	mux.HandleFunc("GET /sse-container", containerHandler)
	mux.HandleFunc("GET /events", eventsHandler)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	log.Printf("HTMX example on http://localhost%s", htmxAddr)
	log.Print("Open the URL in your browser to see live SSE-driven HTML fragment swaps.")
	log.Fatal(http.ListenAndServe( //nolint:gosec // G114: intentional for example server
		htmxAddr,
		mux,
	))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := indexPage().Render(r.Context(), w); err != nil {
		log.Printf("render index page: %v", err)
	}
}

// containerHandler returns the SSE container fragment. The Restart button
// requests this and swaps it in (outerHTML), which re-establishes the SSE
// connection — no JavaScript required.
func containerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := sseContainer().Render(r.Context(), w); err != nil {
		log.Printf("render sse container: %v", err)
	}
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	stream := sse.NewStream(w, r)
	defer func() { _ = stream.Close() }()

	ctx := r.Context()

	for progress := 0; progress <= maxProgress; progress += progressStep {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := stream.Send(sse.Event{
			Event: "progress",
			Data:  renderProgress(ctx, progress),
		}); err != nil {
			return
		}

		if progress < maxProgress {
			select {
			case <-ctx.Done():
				return
			case <-time.After(progressDelay):
			}
		}
	}
}

func renderProgress(ctx context.Context, progress int) string {
	var buf bytes.Buffer

	if err := progressContent(progress).Render(ctx, &buf); err != nil { //nolint:contextcheck // templ takes ctx
		log.Printf("render progress fragment: %v", err)

		return ""
	}

	return buf.String()
}
