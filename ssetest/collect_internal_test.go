package ssetest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
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

// fatalTB routes Cleanup, Helper, and friends to the real test but turns
// Fatalf into a panic carrying the formatted message, so doRequest's fatal
// branches can be asserted with recover instead of killing the test run.
type fatalTB struct {
	*testing.T

	message string
}

func (f *fatalTB) Fatalf(format string, args ...any) {
	f.message = fmt.Sprintf(format, args...)

	panic(f)
}

// requireFatalPanic asserts the callee panicked via fatalTB and that the
// message contains want; the callback form keeps each branch test flat.
func requireFatalPanic(t *testing.T, ftb *fatalTB, want string, callee func()) {
	t.Helper()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected Fatalf to panic, but the call returned")
		}

		if !strings.Contains(ftb.message, want) {
			t.Errorf("Fatalf message %q should contain %q", ftb.message, want)
		}
	}()

	callee()
}

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

func TestCollectN_NonPositiveCountReturnsNil(t *testing.T) {
	t.Parallel()

	// The count guard fires before any server is started, so the handler
	// is never touched.
	if got := CollectN(t, nil, 0); got != nil {
		t.Errorf("CollectN with count 0 = %v, want nil", got)
	}
}

func TestCollectN_FatalsWhenStreamEndsEarly(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: only\n\n")
	})

	ftb := &fatalTB{T: t}

	requireFatalPanic(t, ftb, "read 3 events", func() {
		CollectN(ftb, handler, 3)
	})
}

func TestDoRequest_FatalsOnInvalidMethod(t *testing.T) {
	t.Parallel()

	ftb := &fatalTB{T: t}

	requireFatalPanic(t, ftb, "build", func() {
		doRequest(
			ftb,
			http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
			"BAD METHOD",
			nil,
			"",
			context.Background(),
			nil,
		)
	})
}

func TestDoRequest_FatalsWhenRequestFails(t *testing.T) {
	t.Parallel()

	ftb := &fatalTB{T: t}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	requireFatalPanic(t, ftb, "test server", func() {
		doRequest(
			ftb,
			http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
			http.MethodGet,
			nil,
			"",
			ctx,
			nil,
		)
	})
}

func TestCollectWithTimeout_FatalsWhenHandlerNeverCloses(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		<-r.Context().Done() // hang until the client times out and goes away
	})

	ftb := &fatalTB{T: t}

	requireFatalPanic(t, ftb, "read events within", func() {
		CollectWithTimeout(ftb, handler, 100*time.Millisecond)
	})
}
