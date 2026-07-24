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
