package events_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/snykk/go-rest-boilerplate/internal/platform/events"
)

type created struct{ ID string }

type deleted struct{}

var errHandlerDown = errors.New("handler down")

// fixture is a Bus that logs JSON into logs and records its spans in spans; handlers made by record append to
// calls.
type fixture struct {
	bus   *events.Bus
	logs  *bytes.Buffer
	spans *tracetest.SpanRecorder
	tp    *sdktrace.TracerProvider
	calls []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{logs: &bytes.Buffer{}, spans: tracetest.NewSpanRecorder()}
	f.tp = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(f.spans))
	f.bus = events.NewBus(slog.New(slog.NewJSONHandler(f.logs, nil)), f.tp)
	return f
}

// record returns a handler of created that appends "<name>:<ID>" to f.calls.
func (f *fixture) record(name string) func(context.Context, created) error {
	return func(_ context.Context, event created) error {
		f.calls = append(f.calls, name+":"+event.ID)
		return nil
	}
}

// records returns the log records, each with its values as strings.
func (f *fixture) records(t *testing.T) []map[string]string {
	t.Helper()
	var parsed []map[string]string
	for line := range strings.Lines(f.logs.String()) {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("log line %q is not JSON: %v", line, err)
		}
		fields := make(map[string]string, len(record))
		for key, value := range record {
			fields[key] = fmt.Sprint(value)
		}
		parsed = append(parsed, fields)
	}
	return parsed
}

// Publish runs every handler of the event's exact type in subscription order, each in
// a span of its own under the publisher's, and that a handler's failure or panic is logged without stopping the
// others.
func TestBusPublish(t *testing.T) {
	tCases := []struct {
		name      string
		subscribe func(f *fixture)
		event     any
		wantCalls []string
		wantLogs  []map[string]string
		wantSpans []string
	}{
		{
			name: "every handler of the event's type runs, in the order they subscribed",
			subscribe: func(f *fixture) {
				events.Subscribe(f.bus, "a", f.record("a"))
				events.Subscribe(f.bus, "b", f.record("b"))
			},
			event:     created{ID: "1"},
			wantCalls: []string{"a:1", "b:1"},
			wantSpans: []string{"a", "b"},
		},
		{
			name: "a handler of another type does not run",
			subscribe: func(f *fixture) {
				events.Subscribe(f.bus, "d", func(context.Context, deleted) error {
					f.calls = append(f.calls, "d")
					return nil
				})
			},
			event: created{ID: "1"},
		},
		{
			name:      "an event nobody subscribes to is dropped",
			subscribe: func(*fixture) {},
			event:     created{ID: "1"},
		},
		{
			name:      "a pointer to the subscribed type does not reach the handler",
			subscribe: func(f *fixture) { events.Subscribe(f.bus, "a", f.record("a")) },
			event:     &created{ID: "1"},
		},
		{
			name: "a failing handler is logged and the next one still runs",
			subscribe: func(f *fixture) {
				events.Subscribe(f.bus, "failing", func(context.Context, created) error { return errHandlerDown })
				events.Subscribe(f.bus, "b", f.record("b"))
			},
			event:     created{ID: "1"},
			wantCalls: []string{"b:1"},
			wantLogs: []map[string]string{{
				"level":      "ERROR",
				"msg":        "event handler failed",
				"handler":    "failing",
				"event.type": "events_test.created",
				"error":      errHandlerDown.Error(),
			}},
			wantSpans: []string{"failing", "b"},
		},
		{
			name: "a panicking handler is logged and the next one still runs",
			subscribe: func(f *fixture) {
				events.Subscribe(f.bus, "panicking", func(context.Context, created) error { panic("boom") })
				events.Subscribe(f.bus, "b", f.record("b"))
			},
			event:     created{ID: "1"},
			wantCalls: []string{"b:1"},
			wantLogs: []map[string]string{{
				"level":      "ERROR",
				"msg":        "event handler panicked",
				"handler":    "panicking",
				"event.type": "events_test.created",
				"panic":      "boom",
			}},
			wantSpans: []string{"panicking", "b"},
		},
		{
			name: "each handler runs in a span of its own, child of the publisher's",
			subscribe: func(f *fixture) {
				events.Subscribe(f.bus, "first", f.record("first"))
				events.Subscribe(f.bus, "second", f.record("second"))
			},
			event:     created{ID: "7"},
			wantCalls: []string{"first:7", "second:7"},
			wantSpans: []string{"first", "second"},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.subscribe(f)

			ctx, parent := f.tp.Tracer("test").Start(context.Background(), "parent")
			f.bus.Publish(ctx, tc.event)
			parent.End()

			if !reflect.DeepEqual(f.calls, tc.wantCalls) {
				t.Errorf("calls = %v, want %v", f.calls, tc.wantCalls)
			}
			checkLogs(t, f.records(t), tc.wantLogs)
			checkSpans(t, f.spans.Ended(), parent.SpanContext().SpanID().String(), tc.wantSpans)
		})
	}
}

// checkLogs reports a record count other than len(want), or a record missing one of want's values.
func checkLogs(t *testing.T, got, want []map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("log records = %v, want %d records", got, len(want))
	}
	for i, wantRecord := range want {
		for key, value := range wantRecord {
			if got[i][key] != value {
				t.Errorf("log record %d %s = %q, want %q", i, key, got[i][key], value)
			}
		}
	}
}

// checkSpans reports handler spans other than want, in order, each a child of parentID carrying event.type.
func checkSpans(t *testing.T, ended []sdktrace.ReadOnlySpan, parentID string, want []string) {
	t.Helper()
	var names []string
	for _, span := range ended {
		if span.Name() == "parent" {
			continue
		}
		names = append(names, span.Name())
		if got := span.Parent().SpanID().String(); got != parentID {
			t.Errorf("span %s parent = %s, want %s", span.Name(), got, parentID)
		}
		var eventType string
		for _, keyValue := range span.Attributes() {
			if keyValue.Key == "event.type" {
				eventType = keyValue.Value.AsString()
			}
		}
		if eventType != "events_test.created" {
			t.Errorf("span %s event.type = %q, want %q", span.Name(), eventType, "events_test.created")
		}
	}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("handler spans = %v, want %v", names, want)
	}
}
