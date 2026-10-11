package ssetest

import (
	"net/http"

	"github.com/larsartmann/go-sse"
)

// RequestOption customizes the HTTP request that a Collect* helper sends.
// Options compose: WithPath selects the route, WithHeader adds headers, and
// WithLastEventID sets the reconnection header for replay testing.
type RequestOption func(*requestConfig)

// requestConfig accumulates the applied [RequestOption]s for one request.
type requestConfig struct {
	path    string
	headers http.Header
}

// WithPath targets the request at the given path instead of "/". Use this for
// handlers mounted under a route (e.g., a mux serving "/events") or for
// query-parameter-driven handlers. The path may include its own query string:
//
//	WithPath("/events?filter=alerts")
func WithPath(path string) RequestOption {
	return func(cfg *requestConfig) {
		cfg.path = path
	}
}

// WithHeader adds a request header. Multiple calls with the same key append
// multiple values.
func WithHeader(key, value string) RequestOption {
	return func(cfg *requestConfig) {
		if cfg.headers == nil {
			cfg.headers = make(http.Header)
		}

		cfg.headers.Add(key, value)
	}
}

// WithLastEventID sets the Last-Event-ID request header, simulating a browser
// reconnecting after a dropped connection. Handlers that replay missed events
// (e.g., via go-sse's Replay) respond with everything after the given event ID.
// Use this to E2E test reconnection replay without a real browser.
//
// The id is an [sse.EventID] — the same branded type go-sse writes to the id:
// field and reads back via sse.LastEventIDFromRequest — so an event ID cannot
// be confused with any other string at the call site. Construct test literals
// with sse.NewEventID, or sse.MustParseEventID for values that must validate.
// The zero [sse.EventID] sends no header at all, exactly like a browser's
// initial connection (browsers only send Last-Event-ID after observing an
// id: field).
func WithLastEventID(id sse.EventID) RequestOption {
	return func(cfg *requestConfig) {
		if id.IsZero() {
			return
		}

		if cfg.headers == nil {
			cfg.headers = make(http.Header)
		}

		cfg.headers.Set("Last-Event-ID", id.Get())
	}
}

// targetPath returns the configured path, defaulting to "/" when unset.
func (c *requestConfig) targetPath() string {
	if c.path == "" {
		return "/"
	}

	return c.path
}

// applyRequestOptions folds opts into a fresh config.
func applyRequestOptions(opts []RequestOption) requestConfig {
	var cfg requestConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}
