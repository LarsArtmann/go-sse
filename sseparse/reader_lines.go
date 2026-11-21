package sseparse

import (
	"bytes"
	"errors"
	"io"
)

// crlfLen is the number of bytes consumed when a CR is followed by its LF
// partner: a CRLF pair counts as one end-of-line, not two.
const crlfLen = 2

// splitSSELines is a bufio.SplitFunc that splits a byte stream into lines on
// CR, LF, or CRLF — the three terminators the SSE spec allows (§ 9.2.5:
// "end-of-line = cr lf / cr / lf"). bufio.ScanLines is not sufficient: it only
// splits on LF, so a lone CR would be swallowed into the line instead of
// terminating it.
//
// A CR as the last buffered byte is held back until one more byte arrives (or
// EOF) so a CRLF pair is always recognized as a single terminator.
func splitSSELines(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	for i := range data {
		switch data[i] {
		case '\n':
			return i + 1, data[:i], nil
		case '\r':
			if i+1 < len(data) {
				if data[i+1] == '\n' {
					return i + crlfLen, data[:i], nil // CRLF: one terminator
				}

				return i + 1, data[:i], nil // lone CR
			}

			if atEOF {
				return i + 1, data[:i], nil // lone CR at EOF
			}

			return 0, nil, nil // trailing CR: wait for one more byte
		}
	}

	if atEOF {
		return len(data), data, nil // final unterminated line
	}

	return 0, nil, nil // request more data
}

// stripLeadingBOM wraps r so that exactly one leading UTF-8 byte-order mark is
// removed before parsing. The spec decodes the stream with UTF-8 decode, which
// strips a single leading U+FEFF; the WPT format-bom and format-bom-2 vectors
// pin that a second (mid-stream) BOM is NOT stripped — it poisons the first
// field name, so the line is ignored as an unknown field.
func stripLeadingBOM(r io.Reader) io.Reader {
	return &bomStripReader{r: r} //nolint:exhaustruct_v5 // zero values are correct for the rest
}

// bomStripReader probes the first bytes of the underlying reader once, drops
// them if they form a complete BOM, and otherwise replays them unchanged.
type bomStripReader struct {
	r       io.Reader
	pending []byte // probed bytes that must still be handed out
	checked bool
	err     error // deferred probe error, returned once pending drains
}

func (b *bomStripReader) Read(buf []byte) (int, error) {
	if !b.checked {
		b.probe()
	}

	if len(b.pending) > 0 {
		n := copy(buf, b.pending)
		b.pending = b.pending[n:]

		return n, nil
	}

	if b.err != nil {
		err := b.err
		b.err = nil

		return 0, err
	}

	return b.r.Read(buf) //nolint:wrapcheck // transparent passthrough to the wrapped reader
}

// probe reads up to three bytes and decides whether they are a BOM. Short
// reads are replayed unchanged: only a complete leading BOM is stripped.
// A hit EOF during the probe is deferred and surfaced as io.EOF once any
// replayed bytes have been handed out.
func (b *bomStripReader) probe() {
	b.checked = true

	var head [len(utf8BOM)]byte

	n, err := io.ReadFull(b.r, head[:])
	b.pending = append(b.pending, head[:n]...)

	if n == len(utf8BOM) && bytes.Equal(head[:], utf8BOM[:]) {
		b.pending = b.pending[:0] // drop the BOM
	}

	switch {
	case err == nil:
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		b.err = io.EOF
	default:
		b.err = err
	}
}
