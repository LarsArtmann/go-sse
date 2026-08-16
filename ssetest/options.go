package ssetest

import (
	"net/http"
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
func WithLastEventID(id string) RequestOption {
	return func(cfg *requestConfig) {
		if cfg.headers == nil {
			cfg.headers = make(http.Header)
		}

		cfg.headers.Set("Last-Event-ID", id)
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
