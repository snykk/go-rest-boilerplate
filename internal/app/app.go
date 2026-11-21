// Package app is the composition root: it opens the resources of a process, builds the modules on them,
// and runs the components a binary chooses.
package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/snykk/go-rest-boilerplate/internal/platform/buildinfo"
	"github.com/snykk/go-rest-boilerplate/internal/platform/database"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/lifecycle"
	"github.com/snykk/go-rest-boilerplate/internal/platform/logger"
	"github.com/snykk/go-rest-boilerplate/internal/platform/telemetry"
)

// stopMargin is the time the components get to stop beyond the HTTP shutdown timeout: the HTTP server spends
// it closing the connections of requests that outlived its own timeout, then reports them.
const stopMargin = time.Second

// App is one process: the resources it opened and the modules built on them.
type App struct {
	cfg     Config
	log     *slog.Logger
	handler http.Handler
	closers lifecycle.Closers
}

// New builds the logger and the telemetry, opens the database pool, and builds the modules and the HTTP
// handler, registering a closer for each resource as it opens it. It also installs the telemetry as
// OpenTelemetry's global tracer provider, meter provider and propagator, for code that asks otel for a tracer or
// a meter, and logs the errors OpenTelemetry reports, such as failed exports. When a step fails, New closes what
// it had opened and returns the error.
func New(ctx context.Context, config Config) (*App, error) {
	application := &App{cfg: config}
	if err := application.open(ctx); err != nil {
		return nil, errors.Join(err, application.close(ctx))
	}
	return application, nil
}

func (a *App) open(ctx context.Context) error {
	platform, build := a.cfg.Platform, buildinfo.Get()
	log, err := logger.New(os.Stdout, logger.Options{
		Level:  platform.Log.Level,
		Format: platform.Log.Format,
		Source: platform.Log.Source,
		Attrs: []slog.Attr{
			slog.String("service", platform.Service),
			slog.String("version", build.Version),
			slog.String("env", platform.Env),
		},
	})
	if err != nil {
		return err
	}
	a.log = log
	a.log.InfoContext(ctx, "starting", slog.String("commit", build.Commit))

	providers, err := telemetry.New(ctx, telemetry.Config{
		Service:        platform.Service,
		Version:        build.Version,
		Environment:    platform.Env,
		Endpoint:       platform.Telemetry.Endpoint,
		SampleRatio:    platform.Telemetry.SampleRatio,
		MetricInterval: platform.Telemetry.MetricInterval,
	})
	if err != nil {
		return err
	}
	a.closers.Add("telemetry", providers.Shutdown)
	otel.SetTracerProvider(providers.TracerProvider)
	otel.SetMeterProvider(providers.MeterProvider)
	otel.SetTextMapPropagator(providers.Propagator)
	otel.SetErrorHandler(telemetry.ErrorHandler(a.log))

	pool, err := database.Open(ctx, platform.Database.URL,
		database.WithTelemetry(providers.TracerProvider, providers.MeterProvider))
	if err != nil {
		return err
	}
	a.closers.Add("database", func(context.Context) error {
		pool.Close()
		return nil
	})

	a.handler = NewHandler(a.cfg, Deps{
		Log:       a.log,
		Pool:      pool,
		Telemetry: providers,
	})
	return nil
}

// HTTPServer returns the component that serves the application on cfg.Platform.HTTP.Addr until its context
// ends.
func (a *App) HTTPServer() lifecycle.ComponentFunc {
	config := httpserver.Config{
		Addr:              a.cfg.Platform.HTTP.Addr,
		ReadHeaderTimeout: a.cfg.Platform.HTTP.ReadHeaderTimeout,
		ReadTimeout:       a.cfg.Platform.HTTP.ReadTimeout,
		WriteTimeout:      a.cfg.Platform.HTTP.WriteTimeout,
		IdleTimeout:       a.cfg.Platform.HTTP.IdleTimeout,
		ShutdownTimeout:   a.cfg.Platform.HTTP.ShutdownTimeout,
	}

	return lifecycle.ComponentFunc(func(ctx context.Context) error {
		return httpserver.Run(ctx, config, a.handler, a.log)
	})
}

// Run runs the components with lifecycle.Run. Once ctx ends they get cfg.Platform.HTTP.ShutdownTimeout and one
// second more to stop, so the HTTP server can report requests that outlive its own timeout. Run then closes the
// resources in reverse order within cfg.Platform.HTTP.ShutdownTimeout, and returns the components' error joined
// with any closing error.
func (a *App) Run(ctx context.Context, components ...lifecycle.Component) error {
	err := lifecycle.Run(ctx, a.cfg.Platform.HTTP.ShutdownTimeout+stopMargin, components...)
	return errors.Join(err, a.close(ctx))
}

func (a *App) close(ctx context.Context) error {
	closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.cfg.Platform.HTTP.ShutdownTimeout)
	defer cancel()
	return a.closers.Close(closeCtx)
}
