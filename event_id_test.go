package sse_test

import (
	"bytes"
	"testing"

	"github.com/larsartmann/go-sse"
)

func TestParseEventID_Valid(t *testing.T) {
	t.Parallel()

	cases := []string{"", "42", "evt-99", "550e8400-e29b-41d4-a716-446655440000"}

	for _, input := range cases {
		id, err := sse.ParseEventID(input)
		if err != nil {
			t.Errorf("ParseEventID(%q): unexpected error: %v", input, err)
		}

		if id.Get() != input {
			t.Errorf("ParseEventID(%q): got %q", input, id.Get())
		}
	}
}

func TestParseEventID_RejectsNewlines(t *testing.T) {
	t.Parallel()

	cases := []string{"evt\n42", "evt\r42", "evt\r\n42", "\n"}

	for _, input := range cases {
		_, err := sse.ParseEventID(input)
		if err == nil {
			t.Errorf("ParseEventID(%q): expected error", input)
		}
	}
}

func TestNewEventID(t *testing.T) {
	t.Parallel()

	id := sse.NewEventID("evt-1")
	if id.Get() != "evt-1" {
		t.Errorf("got %q", id.Get())
	}
}

func TestEventID_IsZero(t *testing.T) {
	t.Parallel()

	if !sse.NewEventID("").IsZero() {
		t.Error("empty EventID should be zero")
	}

	if sse.NewEventID("x").IsZero() {
		t.Error("non-empty EventID should not be zero")
	}
}

func TestMustParseEventID_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()

	sse.MustParseEventID("bad\nvalue")
}

func TestMustParseEventID_Valid(t *testing.T) {
	t.Parallel()

	id := sse.MustParseEventID("evt-42")
	if id.Get() != "evt-42" {
		t.Errorf("got %q, want evt-42", id.Get())
	}
}

func TestWriteRetry(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := sse.WriteRetry(&buf, 3000)
	if err != nil {
		t.Fatalf("WriteRetry: %v", err)
	}

	want := "retry: 3000\n\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteRetry_WriteError(t *testing.T) {
	t.Parallel()

	err := sse.WriteRetry(&errorWriter{}, 3000)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEventID_Unicode(t *testing.T) {
	t.Parallel()

	cases := []string{
		"\u65e5\u672c\u8a9e",
		"\U0001f31e",
		"caf\u00e9-m\u00fcnchen-123",
	}

	for _, input := range cases {
		id, err := sse.ParseEventID(input)
		if err != nil {
			t.Errorf("ParseEventID(%q): unexpected error: %v", input, err)
		}

		if id.Get() != input {
			t.Errorf("ParseEventID(%q): got %q", input, id.Get())
		}
	}
}
