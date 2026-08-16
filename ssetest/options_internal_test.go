package ssetest

import "testing"

func TestRequestConfigTargetPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "empty defaults to root", path: "", want: "/"},
		{name: "plain path", path: "/events", want: "/events"},
		{name: "path with query", path: "/events?filter=alerts", want: "/events?filter=alerts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := requestConfig{path: tt.path}
			if got := cfg.targetPath(); got != tt.want {
				t.Errorf("targetPath(): got %q, want %q", got, tt.want)
			}
		})
	}
}
