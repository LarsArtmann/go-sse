// Package main implements a DataStar-compatible SSE server using go-sse.
//
// Demonstrates the full go-sse feature set through a live activity feed:
//   - Broadcaster: multi-client fan-out (open multiple tabs to see it)
//   - SubscriberCount: live count via OnSubscribe/OnUnsubscribe callbacks
//   - SubscribeFilter: "alerts only" view via ?filter=alerts query param
//   - EventStore + Replay: missed events replayed on reconnect
//   - Heartbeat: keeps connections alive through proxies
//   - KeyedLines/SendLines: DataStar keyed-data-line wire format
//
// Run: go run ./example/datastar/
// Then: open http://localhost:8765 in your browser
package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

// Named magic-number constants for the random message generators.
// These make the template strings in producer.go self-documenting.
const (
	nodeCount       = 20
	serviceCount    = 50
	endpointCount   = 10
	deployCount     = 999
	sessionCount    = 9999
	taskCount       = 500
	versionMinor    = 9
	versionPatch    = 9
	maxBuildSeconds = 30
)

const (
	datastarAddr    = ":8765"
	emitInterval    = 2 * time.Second
	heartbeatEvery  = 15 * time.Second
	maxStoredEvents = 50
	shutdownTimeout = 5 * time.Second
	readHeaderLimit = 5 * time.Second

	// idleTimeout is a generous idle timeout for long-lived SSE connections.
	// ReadTimeout/WriteTimeout would kill SSE connections, so we use
	// IdleTimeout instead which only applies to idle keep-alive connections.
	idleTimeout = 5 * time.Minute
)

//go:embed all:static
var staticFiles embed.FS

func main() {
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("static sub FS: %v", err)
	}

	server := newActivityServer()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go server.startProducer(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", server.indexHandler)
	mux.HandleFunc("GET /events", server.eventsHandler)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	//nolint:exhaustruct // example server, most fields are intentionally default
	httpServer := &http.Server{
		Addr:              datastarAddr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderLimit,
		IdleTimeout:       idleTimeout,
	}

	go func() {
		log.Printf("DataStar example on http://localhost%s", datastarAddr)
		log.Print("Open the URL in multiple browser tabs to see real-time fan-out.")

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Print("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Graceful shutdown sequence:
	// 1. HTTP server drains active requests (including SSE connections)
	// 2. Broadcaster drains subscriber buffers, then closes channels
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http server shutdown: %v", err)
	}

	if err := server.broadcaster.Shutdown(shutdownCtx); err != nil {
		log.Printf("broadcaster shutdown: %v", err)
	}
}
