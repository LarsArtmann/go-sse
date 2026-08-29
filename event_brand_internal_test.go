// In-package test (package sse, not sse_test): eventBrand and its Name method
// are unexported, and this is the only symbol that requires internal access.

package sse

import (
	"testing"

	brandid "github.com/larsartmann/go-branded-id"
)

func TestEventBrandName(t *testing.T) {
	t.Parallel()

	if got := (eventBrand{}).Name(); got != "SSEEvent" {
		t.Errorf("eventBrand.Name() = %q, want %q", got, "SSEEvent")
	}
}

func TestEventBrandSatisfiesBrandNamer(t *testing.T) {
	t.Parallel()

	var _ brandid.BrandNamer = eventBrand{}
}

func TestEventBrandNameWiredIntoBrandid(t *testing.T) {
	t.Parallel()

	// Name() exists so brandid's String/GoString/ValidateID diagnostics render
	// "SSEEvent" instead of the raw type name. Pin that wiring.
	if got := brandid.BrandName[eventBrand](); got != "SSEEvent" {
		t.Errorf("brandid.BrandName[eventBrand]() = %q, want %q", got, "SSEEvent")
	}
}
