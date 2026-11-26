// Package telemetry sets up OpenTelemetry: a tracer provider that gives every request a trace ID, a meter
// provider, W3C trace context propagation, and push over OTLP/HTTP when an endpoint is configured.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Config configures New.
type Config struct {
	// Service, Version and Environment describe the process on every span and metric, with the host name and
	// a service.instance.id of its own. OTEL_RESOURCE_ATTRIBUTES adds attributes and may set
	// service.instance.id, but does not replace these three.
	Service, Version, Environment string
	// Endpoint is the OTLP/HTTP base URL of a collector, such as http://collector:4318. Spans go to
	// Endpoint/v1/traces and metrics to Endpoint/v1/metrics. Empty pushes nothing.
	Endpoint string
	// SampleRatio is the share of new traces recorded, from 0 to 1. It does not apply to metrics.
	SampleRatio float64
	// MetricInterval is how often metrics are pushed; 0 means every 60 seconds.
	MetricInterval time.Duration
}

// Providers are the OpenTelemetry providers of one process.
type Providers struct {
	// TracerProvider starts spans. Every span gets a trace ID; a new trace is recorded at Config.SampleRatio,
	// and a trace a caller sampled is always recorded.
	TracerProvider trace.TracerProvider
	// MeterProvider creates the instruments that record metrics. With an endpoint it also records the Go
	// runtime's metrics: goroutines, memory and garbage collection. It drops server.address and server.port
	// from the http.server.* metrics, because a client sets them through the Host header and each new value
	// would be a new series. Without an endpoint it records nothing.
	MeterProvider metric.MeterProvider
	// Propagator reads and writes the W3C traceparent, tracestate and baggage headers.
	Propagator propagation.TextMapPropagator
	shutdown   []func(context.Context) error
}

// New returns the providers of a process described by config. With an endpoint, finished spans are pushed in
// batches and metrics every config.MetricInterval; the OTEL_EXPORTER_OTLP_* variables of the process environment,
// such as OTEL_EXPORTER_OTLP_HEADERS, still apply. It returns an error when OTEL_RESOURCE_ATTRIBUTES is
// malformed or an exporter cannot be built.
func New(ctx context.Context, config Config) (*Providers, error) {
	serviceResource, err := resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(attribute.String("service.instance.id", uuid.NewString())),
		resource.WithFromEnv(),
		resource.WithAttributes(
			attribute.String("service.name", config.Service),
			attribute.String("service.version", config.Version),
			attribute.String("deployment.environment.name", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("describe the service for telemetry: %w", err)
	}

	tracerOptions := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(serviceResource),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(config.SampleRatio))),
	}
	if config.Endpoint != "" {
		exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(signalURL(config.Endpoint, "traces")))
		if err != nil {
			return nil, fmt.Errorf("create the OTLP trace exporter: %w", err)
		}
		tracerOptions = append(tracerOptions, sdktrace.WithBatcher(exporter))
	}
	tracerProvider := sdktrace.NewTracerProvider(tracerOptions...)
	providers := &Providers{
		TracerProvider: tracerProvider,
		MeterProvider:  noop.NewMeterProvider(),
		Propagator:     propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
		shutdown:       []func(context.Context) error{tracerProvider.Shutdown},
	}
	if config.Endpoint == "" {
		return providers, nil
	}

	meterProvider, err := newMeterProvider(ctx, config, serviceResource)
	if err != nil {
		return nil, errors.Join(err, tracerProvider.Shutdown(ctx))
	}
	providers.MeterProvider = meterProvider
	providers.shutdown = append(providers.shutdown, meterProvider.Shutdown)
	return providers, nil
}

// Shutdown pushes the spans and metrics not pushed yet and stops the providers. It gives up when ctx ends.
func (p *Providers) Shutdown(ctx context.Context) error {
	var failures []error
	for _, shutdown := range p.shutdown {
		failures = append(failures, shutdown(ctx))
	}
	return errors.Join(failures...)
}

// ErrorHandler returns an otel.ErrorHandler that logs each error OpenTelemetry reports, such as a failed export
// to the collector, to log at WARN. Installed with otel.SetErrorHandler, it replaces OpenTelemetry's default,
// which prints to the standard library's log package.
func ErrorHandler(log *slog.Logger) otel.ErrorHandler {
	return otel.ErrorHandlerFunc(func(err error) {
		log.WarnContext(context.Background(), "telemetry error", slog.String("error", err.Error()))
	})
}

func newMeterProvider(ctx context.Context, config Config, serviceResource *resource.Resource) (*sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(signalURL(config.Endpoint, "metrics")))
	if err != nil {
		return nil, fmt.Errorf("create the OTLP metric exporter: %w", err)
	}
	var readerOptions []sdkmetric.PeriodicReaderOption
	if config.MetricInterval > 0 {
		readerOptions = append(readerOptions, sdkmetric.WithInterval(config.MetricInterval))
	}
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(serviceResource),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, readerOptions...)),
		sdkmetric.WithView(sdkmetric.NewView(
			sdkmetric.Instrument{Name: "http.server.*"},
			sdkmetric.Stream{AttributeFilter: attribute.NewDenyKeysFilter("server.address", "server.port")},
		)),
	)

	if err := runtime.Start(runtime.WithMeterProvider(provider)); err != nil {
		return nil, errors.Join(fmt.Errorf("record the Go runtime metrics: %w", err), provider.Shutdown(ctx))
	}
	return provider, nil
}

func signalURL(endpoint, signal string) string {
	return strings.TrimSuffix(endpoint, "/") + "/v1/" + signal
}
