package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// durationRoutes returns the http.route of each data point of http.server.request.duration in reader, "" for a
// data point without one.
func durationRoutes(t *testing.T, reader *sdkmetric.ManualReader) []string {
	t.Helper()
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("Collect = %v", err)
	}
	var routes []string
	for _, scope := range collected.ScopeMetrics {
		for _, metric := range scope.Metrics {
			histogram, ok := metric.Data.(metricdata.Histogram[float64])
			if metric.Name != "http.server.request.duration" || !ok {
				continue
			}
			for _, point := range histogram.DataPoints {
				route, _ := point.Attributes.Value("http.route")
				routes = append(routes, route.AsString())
			}
		}
	}
	return routes
}

// Instrument gives each request a server span that continues the caller's trace and records its duration. The
// Router names the span, and labels the duration, after the route the request matched. A request no route
// matches keeps its method as the span name and no route label, and a request to a skipped path is not
// recorded.
func TestInstrument(t *testing.T) {
	const callerTrace = "0af7651916cd43dd8448eb211c80319c"
	tCases := []struct {
		name          string
		target        string
		parent        bool // the request carries the caller's traceparent
		wantSpans     int
		wantName      string
		wantRoute     string   // the http.route attribute; "" means none
		wantDurations []string // the http.route of each recorded duration
	}{
		{
			name:          "a matched request is a span named after its route, in the caller's trace",
			target:        "/v1/articles/42",
			parent:        true,
			wantSpans:     1,
			wantName:      "GET /v1/articles/{id}",
			wantRoute:     "/v1/articles/{id}",
			wantDurations: []string{"/v1/articles/{id}"},
		},
		{
			name:          "an unmatched request is a span named after its method, without a route label",
			target:        "/nope",
			wantSpans:     1,
			wantName:      "GET",
			wantDurations: []string{""},
		},
		{name: "a skipped path is not recorded", target: "/health"},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			spans := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans))
			reader := sdkmetric.NewManualReader()
			meters := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			var inHandler trace.SpanContext
			router := httpserver.NewRouter()
			router.Group("/v1").HandleFunc(http.MethodGet, "/articles/{id}", func(w http.ResponseWriter, r *http.Request) {
				inHandler = trace.SpanContextFromContext(r.Context())
			})
			router.HandleFunc(http.MethodGet, "/health", func(w http.ResponseWriter, r *http.Request) {})
			handler := httpserver.Chain(router.Handler(),
				httpserver.Instrument(provider, meters, propagation.TraceContext{}, "/health"),
				httpserver.RequestID())
			request := httptest.NewRequest(http.MethodGet, tc.target, nil)
			if tc.parent {
				request.Header.Set("traceparent", "00-"+callerTrace+"-b7ad6b7169203331-01")
			}

			handler.ServeHTTP(httptest.NewRecorder(), request)

			if routes := durationRoutes(t, reader); !reflect.DeepEqual(routes, tc.wantDurations) {
				t.Errorf("duration routes = %q, want %q", routes, tc.wantDurations)
			}
			ended := spans.Ended()
			if len(ended) != tc.wantSpans {
				t.Fatalf("spans = %d, want %d", len(ended), tc.wantSpans)
			}
			if tc.wantSpans == 0 {
				return
			}
			span := ended[0]
			if span.Name() != tc.wantName || span.SpanKind() != trace.SpanKindServer {
				t.Errorf("span = %q (%s), want %q (server)", span.Name(), span.SpanKind(), tc.wantName)
			}
			if tc.parent && span.SpanContext().TraceID().String() != callerTrace {
				t.Errorf("trace ID = %s, want the caller's %s", span.SpanContext().TraceID(), callerTrace)
			}
			if tc.parent && inHandler.SpanID() != span.SpanContext().SpanID() {
				t.Errorf("handler span = %s, want the request's span %s", inHandler.SpanID(), span.SpanContext().SpanID())
			}
			var route string
			for _, keyValue := range span.Attributes() {
				if keyValue.Key == attribute.Key("http.route") {
					route = keyValue.Value.AsString()
				}
			}
			if route != tc.wantRoute {
				t.Errorf("http.route = %q, want %q", route, tc.wantRoute)
			}
		})
	}
}
