package sse_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-sse"
)

func TestStream_DoubleCloseSafety(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events", nil)

	stream := sse.NewStream(w, r)

	_ = stream.Close()
	_ = stream.Close() // must not panic
}

func TestStream_SendJSON(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	type payload struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}

	err := stream.SendJSON("update", payload{Name: "go-sse", N: 7})
	if err != nil {
		t.Fatalf("SendJSON: %v", err)
	}

	want := `event: update` + "\n" + `data: {"name":"go-sse","n":7}` + "\n\n"
	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
	}
}

func TestStream_SendJSON_MarshalError(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	// Channels cannot be JSON-marshalled.
	err := stream.SendJSON("bad", make(chan int))
	if err == nil {
		t.Fatal("expected marshal error, got nil")
	}

	if w.Body.Len() != 0 {
		t.Errorf("nothing should be written on marshal failure, got %q", w.Body.String())
	}
}

func TestStream_SendJSON_NilValue(t *testing.T) {
	t.Parallel()

	stream, w := newTestStream(t)

	err := stream.SendJSON("clear", nil)
	if err != nil {
		t.Fatalf("SendJSON(nil): %v", err)
	}

	want := "event: clear\ndata: null\n\n"
	if w.Body.String() != want {
		t.Errorf("got %q, want %q", w.Body.String(), want)
	}
}
