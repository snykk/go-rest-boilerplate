// Package lifecycle runs the long-lived parts of a process together and closes its resources in order.
package lifecycle

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrStopTimeout reports components that did not stop within Run's timeout.
var ErrStopTimeout = errors.New("components did not stop within the timeout")

// Component is a part of a process that runs as long as the process does, such as an HTTP server or a
// queue consumer.
type Component interface {
	// Run runs until ctx ends, then stops and returns. Returning earlier, with or without an error, means
	// the component stopped on its own.
	Run(ctx context.Context) error
}

// ComponentFunc lets a function be a Component.
type ComponentFunc func(ctx context.Context) error

// Run calls f with ctx.
func (f ComponentFunc) Run(ctx context.Context) error { return f(ctx) }

// Run runs the components together until ctx ends or one of them returns. It then cancels the others and
// waits up to timeout for all of them. It returns the first error a component returned, joined with
// ErrStopTimeout when some did not stop in time; it returns nil when every component stopped without an
// error. With no components, Run waits for ctx to end.
func Run(ctx context.Context, timeout time.Duration, components ...Component) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		running  sync.WaitGroup
		mutex    sync.Mutex
		firstErr error
	)
	for _, component := range components {
		running.Go(func() {
			defer cancel()
			if err := component.Run(runCtx); err != nil {
				mutex.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mutex.Unlock()
			}
		})
	}

	stopped := make(chan struct{})
	go func() {
		running.Wait()
		close(stopped)
	}()

	<-runCtx.Done()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	var late error
	select {
	case <-stopped:
	case <-timer.C:
		late = ErrStopTimeout
	}

	mutex.Lock()
	defer mutex.Unlock()
	return errors.Join(firstErr, late)
}
