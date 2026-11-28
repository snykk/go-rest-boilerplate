//go:build integration

package database_test

import (
	"slices"
	"strings"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/snykk/go-rest-boilerplate/internal/platform/database"
	"github.com/snykk/go-rest-boilerplate/internal/testenv"
)

// metricNames returns the names of the metrics reader has collected.
func metricNames(t *testing.T, reader *sdkmetric.ManualReader) map[string]bool {
	t.Helper()
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("Collect = %v", err)
	}
	names := map[string]bool{}
	for _, scope := range collected.ScopeMetrics {
		for _, metric := range scope.Metrics {
			names[metric.Name] = true
		}
	}
	return names
}

// operationNames returns the db.operation.name of each data point of db.client.operation.duration in reader
// that has one.
func operationNames(t *testing.T, reader *sdkmetric.ManualReader) []string {
	t.Helper()
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("Collect = %v", err)
	}
	var operations []string
	for _, scope := range collected.ScopeMetrics {
		for _, metric := range scope.Metrics {
			histogram, ok := metric.Data.(metricdata.Histogram[float64])
			if metric.Name != "db.client.operation.duration" || !ok {
				continue
			}
			for _, point := range histogram.DataPoints {
				if operation, ok := point.Attributes.Value("db.operation.name"); ok {
					operations = append(operations, operation.AsString())
				}
			}
		}
	}
	return operations
}

// WithTelemetry makes the pool record a client span for each query, in the trace of the query's context, named
// after the query's SQL operation and without its arguments. It also records each query's duration, labelled
// with the same operation, and the pool's statistics.
func TestWithTelemetry(t *testing.T) {
	const secret = "s3cret-value"
	tCases := []struct {
		name     string
		sql      string
		wantName string
	}{
		{name: "a select is a span named SELECT", sql: "SELECT length($1::text)", wantName: "SELECT"},
		{
			name:     "a statement with a CTE is named after its first keyword",
			sql:      "WITH v AS (SELECT $1::text AS s) SELECT length(s) FROM v",
			wantName: "WITH",
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			dsn := testenv.Postgres(t, nil).Config().ConnString()
			spans := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans))
			reader := sdkmetric.NewManualReader()
			meters := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			pool, err := database.Open(t.Context(), dsn, database.WithTelemetry(provider, meters))
			if err != nil {
				t.Fatalf("Open = %v", err)
			}
			t.Cleanup(pool.Close)
			ctx, request := provider.Tracer("test").Start(t.Context(), "request")

			var result int
			err = pool.QueryRow(ctx, tc.sql, secret).Scan(&result)
			request.End()

			if err != nil {
				t.Fatalf("query = %v", err)
			}
			var query sdktrace.ReadOnlySpan
			for _, span := range spans.Ended() {
				if span.Name() == tc.wantName {
					query = span
				}
			}
			if query == nil {
				t.Fatalf("no span named %q among %d", tc.wantName, len(spans.Ended()))
			}
			if query.SpanKind() != trace.SpanKindClient {
				t.Errorf("kind = %s, want client", query.SpanKind())
			}
			if query.SpanContext().TraceID() != request.SpanContext().TraceID() {
				t.Errorf("trace ID = %s, want the caller's %s", query.SpanContext().TraceID(), request.SpanContext().TraceID())
			}
			names := metricNames(t, reader)
			for _, want := range []string{"db.client.operation.duration", "pgxpool.acquired_connections", "pgxpool.max_connections"} {
				if !names[want] {
					t.Errorf("metric %s missing among %v", want, names)
				}
			}
			if operations := operationNames(t, reader); !slices.Contains(operations, tc.wantName) {
				t.Errorf("db.client.operation.duration operations = %q, want %q among them", operations, tc.wantName)
			}
			for _, keyValue := range query.Attributes() {
				if strings.Contains(keyValue.Value.String(), secret) {
					t.Errorf("attribute %s = %q, want no query argument recorded", keyValue.Key, keyValue.Value.String())
				}
			}
		})
	}
}
