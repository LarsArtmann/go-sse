package sse

// ContentType is the HTTP content type for Server-Sent Events.
const ContentType = "text/event-stream"

// Common SSE event names. Consumers are free to use custom event names —
// these constants reduce magic strings for the most common patterns.
const (
	// EventConnected is the conventional event name for the initial
	// connection-acknowledgement event sent when an SSE stream opens.
	EventConnected = "connected"

	// EventHeartbeat is the conventional event name for heartbeat/ping
	// events. Note: [Stream.Heartbeat] sends comment-frame pings (not named
	// events); this constant is for consumers who prefer named heartbeats.
	EventHeartbeat = "heartbeat"
)
