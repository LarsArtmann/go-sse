package ssetest_test

import (
	"io"
)

// chunkedReader serves its data in fixed-size chunks, one Read per chunk,
// simulating how a network connection fragments an SSE stream across TCP
// segments. size < 1 behaves like a one-byte reader.
type chunkedReader struct {
	data []byte
	off  int
	size int
}

func (r *chunkedReader) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}

	if r.size < 1 {
		r.size = 1
	}

	n := copy(p, r.data[r.off:min(r.off+r.size, len(r.data))])
	r.off += n

	return n, nil
}
