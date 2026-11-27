package httpserver

import (
	"net/http"
	"slices"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Instrument records every request with OpenTelemetry. It starts a server span with tracerProvider and puts it
// in the request context; the span continues the caller's trace when propagator finds one in the headers, such
// as a W3C traceparent. With meterProvider it records the request's duration and body sizes, labelled with
// method, status, scheme and protocol. They are also labelled with server.address and server.port from the
// Host header, which the client chooses, so meterProvider must drop those two, as the one telemetry.New
// returns does. The span is named after the request method, and the metrics carry no route, until a Router
// names and labels them after the route the request matched. Requests to the paths in skip, such as health
// probes, are not recorded.
func Instrument(tracerProvider trace.TracerProvider, meterProvider metric.MeterProvider,
	propagator propagation.TextMapPropagator, skip ...string) Middleware {
	return otelhttp.NewMiddleware("http.server",
		otelhttp.WithTracerProvider(tracerProvider),
		otelhttp.WithMeterProvider(meterProvider),
		otelhttp.WithPropagators(propagator),
		otelhttp.WithSpanNameFormatter(spanName),
		otelhttp.WithFilter(func(r *http.Request) bool { return !slices.Contains(skip, r.URL.Path) }),
	)
}

func spanName(_ string, r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	return r.Method
}

func labelRoute(method, route string, handler http.Handler) http.Handler {
	name, label := method+" "+route, attribute.String("http.route", route)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetName(name)
		span.SetAttributes(label)
		if labeler, ok := otelhttp.LabelerFromContext(r.Context()); ok {
			labeler.Add(label)
		}
		handler.ServeHTTP(w, r)
	})
}
