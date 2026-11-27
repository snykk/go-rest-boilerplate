package telemetry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
	collectormetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/protobuf/proto"

	"github.com/snykk/go-rest-boilerplate/internal/platform/telemetry"
)

// sampledParent is a W3C traceparent header of a caller that sampled its trace.
const sampledParent = "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"

// collector is an OTLP/HTTP receiver run by a test. It counts the trace exports and keeps the metrics of the
// metric exports.
type collector struct {
	URL     string
	traces  atomic.Int32
	mu      sync.Mutex
	metrics []*metricspb.ResourceMetrics
}

func newCollector(t *testing.T) *collector {
	t.Helper()
	receiver := &collector{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		switch r.URL.Path {
		case "/v1/traces":
			receiver.traces.Add(1)
		case "/v1/metrics":
			var export collectormetricspb.ExportMetricsServiceRequest
			body, err := io.ReadAll(r.Body)
			if err == nil {
				err = proto.Unmarshal(body, &export)
			}
			if err != nil {
				t.Errorf("read a metric export: %v", err)
				return
			}
			receiver.mu.Lock()
			receiver.metrics = append(receiver.metrics, export.GetResourceMetrics()...)
			receiver.mu.Unlock()
		}
	}))
	t.Cleanup(server.Close)
	receiver.URL = server.URL
	return receiver
}

// pushed reports whether a metric export arrived.
func (c *collector) pushed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.metrics) > 0
}

// resource returns the attributes of the resource the first metric export described.
func (c *collector) resource(t *testing.T) map[string]string {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.metrics) == 0 {
		t.Fatal("no metric export arrived")
	}
	return attrs(c.metrics[0].GetResource().GetAttributes())
}

// points returns the attributes of every data point of the metrics whose names start with prefix, by metric
// name.
func (c *collector) points(prefix string) map[string][]map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	points := map[string][]map[string]string{}
	for _, exported := range c.metrics {
		for _, scope := range exported.GetScopeMetrics() {
			for _, metric := range scope.GetMetrics() {
				if !strings.HasPrefix(metric.GetName(), prefix) {
					continue
				}
				for _, point := range metric.GetHistogram().GetDataPoints() {
					points[metric.GetName()] = append(points[metric.GetName()], attrs(point.GetAttributes()))
				}
				for _, point := range metric.GetSum().GetDataPoints() {
					points[metric.GetName()] = append(points[metric.GetName()], attrs(point.GetAttributes()))
				}
			}
		}
	}
	return points
}

// attrs returns keyValues as a map; a value that is not a string, such as a port, maps to its text form.
func attrs(keyValues []*commonpb.KeyValue) map[string]string {
	values := map[string]string{}
	for _, keyValue := range keyValues {
		if text, ok := keyValue.GetValue().GetValue().(*commonpb.AnyValue_StringValue); ok {
			values[keyValue.GetKey()] = text.StringValue
		} else {
			values[keyValue.GetKey()] = keyValue.GetValue().String()
		}
	}
	return values
}

// New gives every span a trace ID and records new traces at the sample ratio, following a caller that sampled
// its trace. With an endpoint it pushes finished spans to <endpoint>/v1/traces and metrics to
// <endpoint>/v1/metrics over OTLP/HTTP, metrics whatever the sample ratio. Shutdown flushes both.
func TestNew(t *testing.T) {
	tCases := []struct {
		name        string
		ratio       float64
		endpoint    bool // push to a collector run by the test
		parent      bool // the span continues sampledParent, read through the propagator
		wantSampled bool
		wantTraces  bool
		wantMetrics bool
	}{
		{name: "without an endpoint a span still gets a trace ID and nothing is pushed", ratio: 1, wantSampled: true},
		{
			name:        "with an endpoint spans and metrics are pushed",
			ratio:       1,
			endpoint:    true,
			wantSampled: true,
			wantTraces:  true,
			wantMetrics: true,
		},
		{name: "a ratio of 0 records no new trace but still pushes metrics", endpoint: true, wantMetrics: true},
		{
			name:        "a caller's sampled trace is followed whatever the ratio",
			parent:      true,
			endpoint:    true,
			wantSampled: true,
			wantTraces:  true,
			wantMetrics: true,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			receiver := newCollector(t)
			config := telemetry.Config{
				Service:     "articles",
				Version:     "v1.2.3",
				Environment: "test",
				SampleRatio: tc.ratio,
			}
			if tc.endpoint {
				config.Endpoint = receiver.URL
			}
			providers, err := telemetry.New(t.Context(), config)
			if err != nil {
				t.Fatalf("New = %v", err)
			}
			ctx := context.Background()
			if tc.parent {
				ctx = providers.Propagator.Extract(ctx, propagation.MapCarrier{"traceparent": sampledParent})
			}
			calls, err := providers.MeterProvider.Meter("test").Int64Counter("test.calls")
			if err != nil {
				t.Fatalf("Int64Counter = %v", err)
			}

			ctx, span := providers.TracerProvider.Tracer("test").Start(ctx, "work")
			calls.Add(ctx, 1)
			span.End()
			err = providers.Shutdown(t.Context())

			if err != nil {
				t.Fatalf("Shutdown = %v", err)
			}
			spanContext := span.SpanContext()
			if !spanContext.TraceID().IsValid() {
				t.Errorf("trace ID = %s, want a valid one", spanContext.TraceID())
			}
			if tc.parent && spanContext.TraceID().String() != "0af7651916cd43dd8448eb211c80319c" {
				t.Errorf("trace ID = %s, want the caller's", spanContext.TraceID())
			}
			if spanContext.IsSampled() != tc.wantSampled {
				t.Errorf("sampled = %t, want %t", spanContext.IsSampled(), tc.wantSampled)
			}
			if pushed := receiver.traces.Load() > 0; pushed != tc.wantTraces {
				t.Errorf("traces pushed = %t, want %t", pushed, tc.wantTraces)
			}
			if pushed := receiver.pushed(); pushed != tc.wantMetrics {
				t.Errorf("metrics pushed = %t, want %t", pushed, tc.wantMetrics)
			}
		})
	}
}

// New describes the process, on everything it pushes, with the configured service, version and environment,
// the host name and an instance ID of its own. OTEL_RESOURCE_ATTRIBUTES adds attributes and may set the
// instance ID, but never replaces the configured service, version or environment; a malformed one is refused.
func TestNewResource(t *testing.T) {
	host, err := os.Hostname()
	if err != nil {
		t.Fatalf("os.Hostname = %v", err)
	}
	tCases := []struct {
		name            string
		env             string // OTEL_RESOURCE_ATTRIBUTES
		want            map[string]string
		wantOwnInstance bool // service.instance.id is set and differs from the one another New call gives
		wantErr         bool
	}{
		{
			name: "the configured service, the host and an instance ID of its own describe the process",
			want: map[string]string{
				"service.name":                "articles",
				"service.version":             "v1.2.3",
				"deployment.environment.name": "test",
				"host.name":                   host,
			},
			wantOwnInstance: true,
		},
		{
			name: "attributes from the environment are added and the configured ones win",
			env:  "team=blog,service.name=other,service.version=v0,deployment.environment.name=other",
			want: map[string]string{
				"team":                        "blog",
				"service.name":                "articles",
				"service.version":             "v1.2.3",
				"deployment.environment.name": "test",
			},
			wantOwnInstance: true,
		},
		{
			name: "an instance ID from the environment is kept",
			env:  "service.instance.id=pod-7",
			want: map[string]string{"service.instance.id": "pod-7"},
		},
		{name: "a malformed OTEL_RESOURCE_ATTRIBUTES is refused", env: "team", wantErr: true},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OTEL_RESOURCE_ATTRIBUTES", tc.env)
			if tc.wantErr {
				if _, err := telemetry.New(t.Context(), telemetry.Config{Service: "articles"}); err == nil {
					t.Error("New = nil error, want the malformed attributes refused")
				}
				return
			}
			describe := func() map[string]string {
				receiver := newCollector(t)
				providers, err := telemetry.New(t.Context(), telemetry.Config{
					Service:     "articles",
					Version:     "v1.2.3",
					Environment: "test",
					Endpoint:    receiver.URL,
				})
				if err != nil {
					t.Fatalf("New = %v", err)
				}
				if err := providers.Shutdown(t.Context()); err != nil {
					t.Fatalf("Shutdown = %v", err)
				}
				return receiver.resource(t)
			}

			got := describe()

			for key, value := range tc.want {
				if got[key] != value {
					t.Errorf("resource %s = %q, want %q", key, got[key], value)
				}
			}
			if tc.wantOwnInstance {
				id, other := got["service.instance.id"], describe()["service.instance.id"]
				if id == "" || id == other {
					t.Errorf("service.instance.id = %q, and %q from another New call, want a distinct ID", id, other)
				}
			}
		})
	}
}

// New's meter provider records the HTTP server metrics without server.address and server.port, which come from
// the Host header a client sends, so the same requests are one series whatever host they name. The other
// attributes stay.
func TestNewHTTPServerMetrics(t *testing.T) {
	receiver := newCollector(t)
	providers, err := telemetry.New(t.Context(), telemetry.Config{
		Service:  "articles",
		Endpoint: receiver.URL,
	})
	if err != nil {
		t.Fatalf("New = %v", err)
	}
	handler := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), "server",
		otelhttp.WithMeterProvider(providers.MeterProvider))

	for _, host := range []string{"a.example:8080", "b.example:9090"} {
		request := httptest.NewRequest(http.MethodGet, "/articles", nil)
		request.Host = host
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
	if err := providers.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown = %v", err)
	}

	metrics := receiver.points("http.server.")
	if len(metrics["http.server.request.duration"]) == 0 {
		t.Fatalf("http.server metrics = %v, want http.server.request.duration among them", metrics)
	}
	for name, points := range metrics {
		if len(points) != 1 {
			t.Errorf("%s data points = %v, want 1", name, points)
			continue
		}
		point := points[0]
		_, address := point["server.address"]
		_, port := point["server.port"]
		if address || port || point["http.request.method"] != "GET" {
			t.Errorf("%s attributes = %v, want http.request.method GET without server.address or server.port", name, point)
		}
	}
}

// ErrorHandler logs each error OpenTelemetry reports, such as a failed export, at WARN with the error's text.
func TestErrorHandler(t *testing.T) {
	var logs bytes.Buffer
	handler := telemetry.ErrorHandler(slog.New(slog.NewJSONHandler(&logs, nil)))

	handler.Handle(errors.New("failed to upload metrics: connection refused"))

	var record map[string]any
	if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
		t.Fatalf("log %q is not one JSON record: %v", logs.String(), err)
	}
	want := map[string]any{
		"level": "WARN",
		"msg":   "telemetry error",
		"error": "failed to upload metrics: connection refused",
	}
	for key, value := range want {
		if record[key] != value {
			t.Errorf("record %s = %v, want %v", key, record[key], value)
		}
	}
}
