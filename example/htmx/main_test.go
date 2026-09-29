// In-package test (package main): listenAddr reads process env, so the tests
// must run serially — hence the paralleltest opt-out per function.

package main

import "testing"

//nolint:paralleltest // t.Setenv pins the test to serial execution

func TestListenAddr_Default(t *testing.T) {
	t.Setenv("PORT", "")

	if got, want := listenAddr(), ":"+defaultHtmxPort; got != want {
		t.Errorf("listenAddr() = %q, want %q", got, want)
	}
}

//nolint:paralleltest // t.Setenv pins the test to serial execution

func TestListenAddr_PortOverride(t *testing.T) {
	t.Setenv("PORT", "18766")

	if got, want := listenAddr(), ":18766"; got != want {
		t.Errorf("listenAddr() = %q, want %q", got, want)
	}
}
