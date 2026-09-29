// In-package test (package sse): forEachLine is unexported, and the spec § 9.2.5
// end-of-line rules it implements deserve direct pinning rather than coverage
// only through splitLines and the wire conformance corpora.

package sse

import (
	"reflect"
	"testing"
)

func TestForEachLine(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in   string
		want []string
	}{
		"empty yields exactly one empty line": {in: "", want: []string{""}},
		"single line without terminator":      {in: "data: x", want: []string{"data: x"}},
		"LF separated":                        {in: "a\nb", want: []string{"a", "b"}},
		"trailing LF yields no final empty":   {in: "a\n", want: []string{"a"}},
		"CRLF is one terminator":              {in: "a\r\nb", want: []string{"a", "b"}},
		"lone CR is a terminator":             {in: "a\rb", want: []string{"a", "b"}},
		"mixed terminators":                   {in: "a\r\nb\rc\nd", want: []string{"a", "b", "c", "d"}},
		"empty lines between terminators":     {in: "a\n\nb", want: []string{"a", "", "b"}},
		"terminator sandwich":                 {in: "\nx\n", want: []string{"", "x"}},
		"lone CRLF yields one empty line":     {in: "\r\n", want: []string{""}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got []string

			forEachLine(tt.in, func(line string) { got = append(got, line) })

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("forEachLine(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
