// Package id generates unique identifiers.
package id

import "github.com/google/uuid"

// New returns a new UUIDv7 in its canonical 36-character form. UUIDv7s sort by creation time.
func New() string {
	return uuid.Must(uuid.NewV7()).String()
}

// Generator hands out IDs from New.
type Generator struct{}

// NewID returns a new UUIDv7, as New does.
func (Generator) NewID() string { return New() }
