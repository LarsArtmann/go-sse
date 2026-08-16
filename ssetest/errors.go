package ssetest

// Error codes for ssetest. Each is a stable string accessible via
// [errorfamily.Code], enabling programmatic classification of errors returned
// by the test helpers without string matching on human-readable messages.
const (
	// CodeSSEScanFailed: [ReadEvents] or [ReadNEvents] encountered an I/O
	// error while scanning the SSE response stream.
	CodeSSEScanFailed = "ssetest.sse_scan_failed"
)
