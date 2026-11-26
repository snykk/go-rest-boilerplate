package lifecycle_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/lifecycle"
)

var errBoom = errors.New("boom")

// blocks is a component that runs until its context ends, then closes stopped.
func blocks(stopped chan<- struct{}) lifecycle.Component {
	return lifecycle.ComponentFunc(func(ctx context.Context) error {
		<-ctx.Done()
		close(stopped)
		return nil
	})
}

// returns is a component that returns err after delay, whatever its context does.
func returns(delay time.Duration, err error) lifecycle.Component {
	return lifecycle.ComponentFunc(func(context.Context) error {
		time.Sleep(delay)
		return err
	})
}

// Run stops every component when its context ends or when one component stops, and gives up on components
// that do not stop within the timeout.
func TestRun(t *testing.T) {
	tCases := []struct {
		name        string
		components  func(stopped chan<- struct{}) []lifecycle.Component
		cancelAfter time.Duration // the test cancels Run's context after this; 0 means never
		timeout     time.Duration
		wantErr     error
		wantStopped bool // the blocking component saw the cancellation
		within      time.Duration
	}{
		{
			name: "every component stops when the context ends",
			components: func(stopped chan<- struct{}) []lifecycle.Component {
				return []lifecycle.Component{blocks(stopped), blocks(make(chan struct{}))}
			},
			cancelAfter: 20 * time.Millisecond,
			timeout:     time.Second,
			wantStopped: true,
			within:      time.Second,
		},
		{
			name: "a component that fails stops the others",
			components: func(stopped chan<- struct{}) []lifecycle.Component {
				return []lifecycle.Component{returns(10*time.Millisecond, errBoom), blocks(stopped)}
			},
			timeout:     time.Second,
			wantErr:     errBoom,
			wantStopped: true,
			within:      time.Second,
		},
		{
			name: "a component that returns early stops the others",
			components: func(stopped chan<- struct{}) []lifecycle.Component {
				return []lifecycle.Component{returns(0, nil), blocks(stopped)}
			},
			timeout:     time.Second,
			wantStopped: true,
			within:      time.Second,
		},
		{
			name: "a component that ignores cancellation makes Run give up after the timeout",
			components: func(chan<- struct{}) []lifecycle.Component {
				return []lifecycle.Component{returns(time.Second, nil)}
			},
			cancelAfter: time.Millisecond,
			timeout:     50 * time.Millisecond,
			wantErr:     lifecycle.ErrStopTimeout,
			within:      300 * time.Millisecond,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			stopped := make(chan struct{})
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tc.cancelAfter > 0 {
				time.AfterFunc(tc.cancelAfter, cancel)
			}
			start := time.Now()

			err := lifecycle.Run(ctx, tc.timeout, tc.components(stopped)...)

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Run = %v, want %v", err, tc.wantErr)
			}
			if took := time.Since(start); took > tc.within {
				t.Errorf("Run took %v, want under %v", took, tc.within)
			}
			if tc.wantStopped {
				select {
				case <-stopped:
				default:
					t.Error("the blocking component did not see the cancellation")
				}
			}
		})
	}
}
