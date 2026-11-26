package id_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// New returns a different UUIDv7 on every call.
func TestNew(t *testing.T) {
	first, second := id.New(), id.New()
	parsed, err := uuid.Parse(first)
	if err != nil {
		t.Fatalf("New = %q, not a UUID: %v", first, err)
	}
	if parsed.Version() != 7 {
		t.Errorf("New version = %d, want 7", parsed.Version())
	}
	if first == second {
		t.Errorf("New returned %q twice", first)
	}
	if got := (id.Generator{}).NewID(); len(got) != 36 {
		t.Errorf("Generator.NewID = %q, want a 36-character UUID", got)
	}
}
