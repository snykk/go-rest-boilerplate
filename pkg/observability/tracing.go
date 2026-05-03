package observability

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerName is the import path callers should use when grabbing a
// tracer from the global provider, so every span emitted from this
// project shares the same instrumentation library label.
const TracerName = "github.com/snykk/go-rest-boilerplate"

// TracingConfig drives tracer-provider construction. Zero value gives
// a no-op tracer (sampler=never, no exporter), which is what tests
// and `go run` without OTel env vars get by default.
type TracingConfig struct {
	// ServiceName is set as the resource attribute service.name.
	ServiceName string
	// Environment ends up as deployment.environment.
	Environment string
	// Exporter selects the destination: "stdout" (dev), "otlp" (prod
	// via OTLP/gRPC to OTEL_EXPORTER_OTLP_ENDPOINT), "" (disabled).
	Exporter string
	// SampleRatio is the head sampler ratio (0..1). 0 disables, 1
	// records everything. Production typically sits at 0.01–0.1.
	SampleRatio float64
}

// Shutdown drains buffered spans. Hook this into the server's
// graceful-shutdown sequence so tail spans aren't lost on SIGTERM.
type Shutdown func(context.Context) error

// SetupTracing installs a global tracer provider + W3C trace-context
// propagator and returns a shutdown function. When cfg.Exporter is
// empty, returns a no-op Shutdown so call sites don't have to nil-check.
func SetupTracing(ctx context.Context, cfg TracingConfig) (Shutdown, error) {
	if cfg.Exporter == "" {
		// Tracing disabled — leave the global provider as the OTel
		// noop default so any `tracer.Start(...)` calls just no-op.
		// Propagator is still installed so an upstream traceparent
		// header survives intact through this service.
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := buildExporter(ctx, cfg.Exporter)
	if err != nil {
		return nil, fmt.Errorf("build trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(orDefault(cfg.ServiceName, "go-rest-boilerplate")),
			semconv.DeploymentEnvironment(orDefault(cfg.Environment, "development")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build resource: %w", err)
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(clampRatio(cfg.SampleRatio)))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// Tracer returns a tracer for callers that want to emit manual spans.
// Always pulls from the global provider so SetupTracing's choice
// applies project-wide.
func Tracer() trace.Tracer { return otel.Tracer(TracerName) }

func buildExporter(ctx context.Context, kind string) (sdktrace.SpanExporter, error) {
	switch kind {
	case "stdout":
		// Pretty-print to stdout. Useful in dev to confirm spans are
		// firing without needing an OTel collector running locally.
		return stdouttrace.New(stdouttrace.WithWriter(os.Stdout), stdouttrace.WithPrettyPrint())
	case "otlp":
		// Endpoint, headers, TLS, etc. are read from the standard
		// OTEL_EXPORTER_OTLP_* env vars by otlptracegrpc — that way
		// we don't reinvent config knobs the OTel SDK already owns.
		return otlptracegrpc.New(ctx)
	default:
		return nil, fmt.Errorf("unknown exporter %q (expected stdout|otlp)", kind)
	}
}

func clampRatio(r float64) float64 {
	if r <= 0 {
		return 0
	}
	if r >= 1 {
		return 1
	}
	return r
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
