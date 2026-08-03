package sse_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-sse"
)

// newTestStream builds a Stream backed by an httptest.ResponseRecorder and
// registers stream.Close() as t.Cleanup. Tests that need the underlying
// ResponseRecorder (to assert on the emitted body) take the second return
// value; tests that only need the Stream ignore it.
func newTestStream(t *testing.T) (*sse.Stream, *httptest.ResponseRecorder) {
	t.Helper()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	t.Cleanup(func() { _ = stream.Close() })

	return stream, w
}

// newTestFailingStream builds a Stream whose Write calls return an error.
// Used by tests that exercise write-failure paths. The returned
// *httptest.ResponseRecorder (embedded in the error wrapper) still buffers
// any bytes successfully written before the error, which is useful for
// debugging. Close is registered as t.Cleanup.
func newTestFailingStream(t *testing.T) *sse.Stream {
	t.Helper()

	w := &errorResponseWriter{ResponseWriter: httptest.NewRecorder(), writer: &errorWriter{}}
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)
	t.Cleanup(func() { _ = stream.Close() })

	return stream
}

// errorResponseWriter wraps an errorWriter as http.ResponseWriter so the
// Stream treats the underlying connection as failing on every Write.
type errorResponseWriter struct {
	http.ResponseWriter
	writer *errorWriter
}

func (e *errorResponseWriter) Write(p []byte) (int, error) {
	return e.writer.Write(p)
}

func (e *errorResponseWriter) Flush() {}
