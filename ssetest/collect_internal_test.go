package ssetest

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

// recordingTB captures Errorf calls instead of failing the run, so the
// closeBody error branch can be exercised without a real HTTP round trip.
type tbRecorder struct {
	testing.TB

	errors []string
}

func (r *tbRecorder) Helper() {}

func (r *tbRecorder) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

// errBody is an io.ReadCloser whose Close fails with a fixed error; reads
// delegate to the wrapped reader so closeBody callers can still consume.
type errBody struct {
	io.Reader
	closeErr error
}

func (b *errBody) Close() error { return b.closeErr }

func TestCloseBody_NilErrorIsSilent(t *testing.T) {
	t.Parallel()

	tb := &tbRecorder{}
	closeBody(tb, io.NopCloser(strings.NewReader("data: x\n\n")))

	if len(tb.errors) != 0 {
		t.Errorf("healthy close reported errors: %v", tb.errors)
	}
}

func TestCloseBody_ReportsCloseError(t *testing.T) {
	t.Parallel()

	tb := &tbRecorder{}
	closeBody(tb, &errBody{
		Reader:   strings.NewReader("data: x\n\n"),
		closeErr: errors.New("connection reset during close"),
	})

	if len(tb.errors) != 1 {
		t.Fatalf("expected 1 reported error, got %d: %v", len(tb.errors), tb.errors)
	}

	if !strings.Contains(tb.errors[0], "close response body") ||
		!strings.Contains(tb.errors[0], "connection reset during close") {
		t.Errorf("error message should name the close failure; got %q", tb.errors[0])
	}
}
