package lifecycle_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/lifecycle"
)

// Close runs closers in reverse order, reports each failure by name, and stops waiting when its context ends.
func TestClosersClose(t *testing.T) {
	errB := errors.New("b failed")
	record := func(name string, ran *[]string) func(context.Context) error {
		return func(context.Context) error { *ran = append(*ran, name); return nil }
	}
	tCases := []struct {
		name        string
		add         func(closers *lifecycle.Closers, ran *[]string)
		ctxTimeout  time.Duration // 0 means no deadline
		wantRan     []string
		wantErr     error
		wantMention string // text the error must contain
	}{
		{
			name: "closers run in the reverse order of Add",
			add: func(closers *lifecycle.Closers, ran *[]string) {
				closers.Add("a", record("a", ran))
				closers.Add("b", record("b", ran))
			},
			wantRan: []string{"b", "a"},
		},
		{
			name: "a failure is reported by name and the next closer still runs",
			add: func(closers *lifecycle.Closers, ran *[]string) {
				closers.Add("a", record("a", ran))
				closers.Add("b", func(context.Context) error { return errB })
			},
			wantRan:     []string{"a"},
			wantErr:     errB,
			wantMention: "b: ",
		},
		{
			name: "a closer still running when the context ends is reported and the rest are skipped",
			add: func(closers *lifecycle.Closers, ran *[]string) {
				closers.Add("a", record("a", ran))
				closers.Add("b", func(context.Context) error { time.Sleep(time.Second); return nil })
			},
			ctxTimeout:  50 * time.Millisecond,
			wantErr:     context.DeadlineExceeded,
			wantMention: "b: ",
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			var closers lifecycle.Closers
			var ran []string
			tc.add(&closers, &ran)
			ctx := t.Context()
			if tc.ctxTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tc.ctxTimeout)
				defer cancel()
			}

			err := closers.Close(ctx)

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Close = %v, want %v", err, tc.wantErr)
			}
			if tc.wantMention != "" && (err == nil || !strings.Contains(err.Error(), tc.wantMention)) {
				t.Errorf("Close = %v, want it to mention %q", err, tc.wantMention)
			}
			if !reflect.DeepEqual(ran, tc.wantRan) {
				t.Errorf("ran %v, want %v", ran, tc.wantRan)
			}
		})
	}
}
