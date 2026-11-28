package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"slices"
)

// Closers closes resources in the reverse order they were added. The zero value is ready to use. It is not
// safe for concurrent use.
type Closers struct {
	closers []namedCloser
}

type namedCloser struct {
	name string
	fn   func(ctx context.Context) error
}

// Add records closeFunc as the closer of the resource called name.
func (c *Closers) Add(name string, closeFunc func(ctx context.Context) error) {
	c.closers = append(c.closers, namedCloser{name: name, fn: closeFunc})
}

// Close runs every closer once, in the reverse order of Add. Each failure is reported prefixed with its
// name ("database: …"), and the next closer still runs. When ctx ends while a closer runs, Close reports
// that closer with ctx's error and skips the rest. It returns every report joined, or nil.
func (c *Closers) Close(ctx context.Context) error {
	closers := c.closers
	c.closers = nil

	var failures []error
	for _, closer := range slices.Backward(closers) {
		done := make(chan error, 1)
		go func() { done <- closer.fn(ctx) }()
		select {
		case err := <-done:
			if err != nil {
				failures = append(failures, fmt.Errorf("%s: %w", closer.name, err))
			}
		case <-ctx.Done():
			return errors.Join(append(failures, fmt.Errorf("%s: %w", closer.name, ctx.Err()))...)
		}
	}
	return errors.Join(failures...)
}
