package sse_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/larsartmann/go-sse"
)

// parseDataLines is a minimal SSE data-line extractor for these property
// tests: it returns the payload of every data line in the first frame of the
// wire (one leading space stripped per spec § 9.2.6), stopping at the frame
// terminator. Only data lines are relevant to KeyedLines' contract.
func parseDataLines(t *testing.T, wire string) []string {
	t.Helper()

	var data []string

	for line := range strings.SplitSeq(wire, "\n") {
		switch {
		case strings.HasPrefix(line, "data: "):
			data = append(data, strings.TrimPrefix(line, "data: "))
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(line, "data:"))
		case line == "":
			return data
		}
	}

	return data
}

// TestKeyedLines_WireRoundTrip is the property test: whatever value goes into
// KeyedLines comes back out of the wire as the identical keyed string, one
// "data: key line" per line, with the terminator normalization (CR/CRLF → LF,
// trailing terminator dropped) applied exactly once.
func TestKeyedLines_WireRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		key   string
		value string
	}{
		{"plain html", "elements", "<div>\n  <span>Hello</span>\n</div>"},
		{"single line", "selector", "#feed"},
		{"crlf value", "elements", "<div>\r\n  <span>Hi</span>\r\n</div>"},
		{"lone cr value", "elements", "<div>\r  <span>Hi</span>"},
		{"trailing lf", "elements", "<div>\n"},
		{"empty value", "elements", ""},
		{"empty key", "", "value"},
		{"key with spaces", "my key", "v1\nv2"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			keyed := sse.KeyedLines(tc.key, tc.value)

			var buf bytes.Buffer
			if err := sse.WriteEvent(&buf, sse.Event{Event: "patch", Data: keyed}); err != nil {
				t.Fatalf("write event: %v", err)
			}

			dataLines := parseDataLines(t, buf.String())

			if tc.value == "" {
				// KeyedLines("") = "" → no keyed line; the writer still emits
				// its single empty data line (documented writer contract).
				if len(dataLines) != 1 || dataLines[0] != "" {
					t.Fatalf("empty value wrote %q, want one empty data line", dataLines)
				}

				return
			}

			// Each line of the keyed string must be one data line, verbatim.
			keyedLines := strings.Split(keyed, "\n")
			if len(dataLines) != len(keyedLines) {
				t.Fatalf("data line count: got %d (%q), want %d (keyed %q)",
					len(dataLines), dataLines, len(keyedLines), keyedLines)
			}

			for i := range keyedLines {
				if dataLines[i] != keyedLines[i] {
					t.Fatalf("data line[%d]: got %q, want %q", i, dataLines[i], keyedLines[i])
				}
			}

			// Joining the wire data lines back with \n reconstructs the exact
			// KeyedLines result — the property the DataStar protocol relies on.
			if rejoined := strings.Join(dataLines, "\n"); rejoined != keyed {
				t.Fatalf("rejoined payload: got %q, want %q", rejoined, keyed)
			}
		})
	}
}

// TestSendKeyed_WireRoundTrip proves the Stream path produces the identical
// wire as the raw WriteKeyedLines path: real HTTP through httptest, the
// response body's data lines rejoined exactly reconstruct the KeyedLines
// result.
func TestSendKeyed_WireRoundTrip(t *testing.T) {
	t.Parallel()

	const (
		eventType = "datastar-patch-elements"
		key       = "elements"
		value     = "<div>\n  <span>Hello</span>\n</div>"
	)

	var body bytes.Buffer

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := sse.NewStream(w, r)
		defer func() { _ = stream.Close() }()

		if err := stream.Send(
			sse.Event{Event: eventType, Data: sse.KeyedLines(key, value)},
		); err != nil {
			t.Errorf("send keyed: %v", err)
		}
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if _, err := body.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read body: %v", err)
	}

	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Errorf("content type: got %q", got)
	}

	wantFrame := "event: " + eventType + "\n"

	var frame strings.Builder
	frame.WriteString(wantFrame)
	for line := range strings.SplitSeq(sse.KeyedLines(key, value), "\n") {
		frame.WriteString("data: ")
		frame.WriteString(line)
		frame.WriteString("\n")
	}
	frame.WriteString("\n")
	wantFrame = frame.String()

	if got := body.String(); got != wantFrame {
		t.Fatalf("wire frame:\n got %q\nwant %q", got, wantFrame)
	}

	dataLines := parseDataLines(t, body.String())
	if rejoined := strings.Join(dataLines, "\n"); rejoined != sse.KeyedLines(key, value) {
		t.Fatalf("rejoined: got %q, want %q", rejoined, sse.KeyedLines(key, value))
	}
}

// TestKeyedLines_JoinLinesComposition pins the composition contract from the
// docs: JoinLines of plain keys plus KeyedLines results writes one data line
// per joined line, in order.
func TestKeyedLines_JoinLinesComposition(t *testing.T) {
	t.Parallel()

	data := sse.JoinLines("selector #feed", "mode inner", sse.KeyedLines("elements", "<b>\ni</b>"))

	var buf bytes.Buffer
	if err := sse.WriteEvent(&buf, sse.Event{Event: "patch", Data: data}); err != nil {
		t.Fatalf("write event: %v", err)
	}

	want := []string{"selector #feed", "mode inner", "elements <b>", "elements i</b>"}
	got := parseDataLines(t, buf.String())

	if len(got) != len(want) {
		t.Fatalf("data lines: got %q, want %q", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("data line[%d]: got %q, want %q", i, got[i], want[i])
		}
	}

	// Total frame sanity: 4 data lines, one event line, one blank terminator.
	if n := strings.Count(buf.String(), "\n"); n != len(want)+2 {
		t.Errorf("newline count: got %d, want %d", n, len(want)+2)
	}
}

// TestKeyedLines_LineCountMatchesValue pins the count property for the fuzz
// corpus: KeyedLines never merges, drops, or invents lines relative to the
// terminator-normalized value.
func TestKeyedLines_LineCountMatchesValue(t *testing.T) {
	t.Parallel()

	values := []string{
		"a",
		"a\nb\nc",
		"a\r\nb\rc\nd",
		"\n\na",
		"a\n\n",
		strings.Repeat("x\n", 99) + "y",
	}

	for i, value := range values {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			keyed := sse.KeyedLines("k", value)
			keyedCount := strings.Count(keyed, "\n") + 1

			// Normalize the value the way splitLines does: CR/CRLF → LF, no
			// trailing empty segment from a final terminator.
			normalized := strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
			normalized = strings.TrimSuffix(normalized, "\n")
			valueCount := strings.Count(normalized, "\n") + 1

			if keyedCount != valueCount {
				t.Fatalf("KeyedLines(%q): %d lines, value has %d", value, keyedCount, valueCount)
			}

			if valueCount > 0 && !strings.HasPrefix(strings.SplitN(keyed, "\n", 2)[0], "k ") {
				t.Errorf("first line must carry the key prefix; got %q", keyed)
			}
		})
	}
}
