package ssetest_test

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/larsartmann/go-sse"
	"github.com/larsartmann/go-sse/ssetest"
)

// streamHandler wires a send callback into a real SSE stream, mirroring the
// shape of a production handler.
func streamHandler(send func(*sse.Stream)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		send(stream)
	})
}

func readAll(t *testing.T, r *http.Request) []byte {
	t.Helper()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}

	return data
}

func TestCollect_StreamHandler(t *testing.T) {
	t.Parallel()

	handler := streamHandler(func(stream *sse.Stream) {
		_ = stream.Send(sse.Event{Event: "feed", Data: "hello"})
		_ = stream.Send(sse.Event{Event: "feed", Data: "world", ID: sse.NewEventID("2")})
	})

	events := ssetest.Collect(t, handler)
	ssetest.RequireEventCount(t, events, 2)

	ssetest.RequireEventType(t, events[0], "feed")
	ssetest.RequireData(t, events[0], "hello")
	ssetest.RequireEventID(t, events[1], "2")
}

func TestCollectPost(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}

		if err := json.Unmarshal(readAll(t, r), &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		if body.Name == "" {
			body.Name = "anonymous"
		}

		_ = sse.NewStream(w, r).Send(sse.Event{Event: "greeting", Data: "hi " + body.Name})
	})

	events := ssetest.CollectPost(t, handler, `{"name":"alice"}`)
	ssetest.RequireEventCount(t, events, 1)
	ssetest.RequireData(t, events[0], "hi alice")
}

func TestCollectWithRequest_PutWithBody(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

			return
		}

		_ = sse.NewStream(w, r).Send(sse.Event{Event: "result", Data: "put received"})
	})

	events := ssetest.CollectWithRequest(t, handler, http.MethodPut, nil, "application/json")
	ssetest.RequireEventCount(t, events, 1)
	ssetest.RequireData(t, events[0], "put received")
}

func TestCollectN_StreamingHandler(t *testing.T) {
	t.Parallel()

	handler := streamHandler(func(stream *sse.Stream) {
		for i := range 10 {
			_ = stream.Send(sse.Event{Event: "feed", Data: strconv.Itoa(i)})
		}

		<-stream.Context().Done()
	})

	events := ssetest.CollectN(t, handler, 3)
	ssetest.RequireEventCount(t, events, 3)

	for i, evt := range events {
		ssetest.RequireData(t, evt, strconv.Itoa(i))
	}
}

func TestCollectN_ZeroCount(t *testing.T) {
	t.Parallel()

	events := ssetest.CollectN(t, streamHandler(func(stream *sse.Stream) {
		_ = stream.Send(sse.Event{Data: "x"})
	}), 0)

	if len(events) != 0 {
		t.Errorf("CollectN(0): got %d events, want 0", len(events))
	}
}

func TestCollectN_FewerThanRequested(t *testing.T) {
	t.Parallel()

	handler := streamHandler(func(stream *sse.Stream) {
		_ = stream.Send(sse.Event{Data: "1"})
		_ = stream.Send(sse.Event{Data: "2"})
	})

	events := ssetest.CollectN(t, handler, 10)
	ssetest.RequireEventCount(t, events, 2)
}

func TestCollectWithTimeout(t *testing.T) {
	t.Parallel()

	handler := streamHandler(func(stream *sse.Stream) {
		_ = stream.Send(sse.Event{Event: "feed", Data: "hi"})
	})

	events := ssetest.CollectWithTimeout(t, handler, 5*time.Second)
	ssetest.RequireEventCount(t, events, 1)
	ssetest.RequireData(t, events[0], "hi")
}

func TestCollectWithTimeout_StreamingReturnsPartial(t *testing.T) {
	t.Parallel()

	handler := streamHandler(func(stream *sse.Stream) {
		_ = stream.Send(sse.Event{Event: "feed", Data: "1"})

		<-stream.Context().Done()
	})

	events := ssetest.CollectWithTimeout(t, handler, 200*time.Millisecond)
	ssetest.RequireEventCount(t, events, 1)
}
