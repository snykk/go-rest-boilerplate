package httpserver_test

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

var discard = slog.New(slog.DiscardHandler)

// Serve lets a request in flight finish when its context ends and returns nil. A request that outlives the
// shutdown timeout has its connection closed and its context cancelled, and Serve reports the timeout.
func TestServe(t *testing.T) {
	tCases := []struct {
		name       string
		handle     http.HandlerFunc
		timeout    time.Duration
		wantStatus int   // 0 means the client got no response
		wantErr    error // nil means Serve returned nil
	}{
		{
			name: "a request in flight finishes and Serve returns nil",
			handle: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(200 * time.Millisecond)
				w.WriteHeader(200)
			},
			timeout:    2 * time.Second,
			wantStatus: 200,
		},
		{
			name: "a request that outlives the shutdown timeout is cancelled and Serve reports the timeout",
			handle: func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
				case <-time.After(5 * time.Second):
					w.WriteHeader(200)
				}
			},
			timeout: 100 * time.Millisecond,
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			started := make(chan struct{})
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(started)
				tc.handle(w, r)
			})
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			go func() {
				done <- httpserver.Serve(ctx, listener, httpserver.Config{ShutdownTimeout: tc.timeout}, handler, discard)
			}()
			status := make(chan int, 1)
			go func() {
				response, err := http.Get("http://" + listener.Addr().String())
				if err != nil {
					status <- 0
					return
				}
				response.Body.Close()
				status <- response.StatusCode
			}()
			deadline := time.After(3 * time.Second)

			<-started
			cancel()

			select {
			case got := <-status:
				if got != tc.wantStatus {
					t.Errorf("in-flight request status = %d, want %d", got, tc.wantStatus)
				}
			case <-deadline:
				t.Fatal("the in-flight request was still running 3s after the shutdown began")
			}
			if err := <-done; !errors.Is(err, tc.wantErr) || (tc.wantErr == nil && err != nil) {
				t.Errorf("Serve = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// ctxKey keys the value a ctxRecorder reads from the context of each record.
type ctxKey struct{}

// ctxRecorder is a slog.Handler that keeps, for each record, its message and the ctxKey value of the context
// it was logged with.
type ctxRecorder struct {
	mu      sync.Mutex
	records map[string]any
}

func (h *ctxRecorder) Enabled(context.Context, slog.Level) bool { return true }

func (h *ctxRecorder) Handle(ctx context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records[record.Message] = ctx.Value(ctxKey{})
	return nil
}

func (h *ctxRecorder) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *ctxRecorder) WithGroup(string) slog.Handler { return h }

// Serve logs that it listens and that it shuts down with its context, so both records carry what the context
// holds, such as a trace.
func TestServeLogs(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	recorder := &ctxRecorder{records: map[string]any{}}
	ctx, cancel := context.WithCancel(context.WithValue(t.Context(), ctxKey{}, "process"))

	cancel()
	err = httpserver.Serve(ctx, listener, httpserver.Config{ShutdownTimeout: time.Second}, http.NotFoundHandler(),
		slog.New(recorder))

	if err != nil {
		t.Fatalf("Serve = %v", err)
	}
	want := map[string]any{"listening": "process", "shutting down": "process"}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if !reflect.DeepEqual(recorder.records, want) {
		t.Errorf("context of each record = %v, want %v", recorder.records, want)
	}
}

// Run reports an address it cannot listen on.
func TestRun(t *testing.T) {
	err := httpserver.Run(t.Context(), httpserver.Config{Addr: "256.0.0.1:99999"}, http.NotFoundHandler(), discard)
	if err == nil {
		t.Error("Run = nil, want the listen error")
	}
}
