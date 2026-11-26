// Package database opens the Postgres connection pool and defines Querier, the pool or transaction that
// gateways run their SQL on.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Option changes how Open builds the pool.
type Option func(poolSettings *settings)

type settings struct {
	tracerProvider trace.TracerProvider
	meterProvider  metric.MeterProvider
}

// WithTelemetry makes the pool record each query with OpenTelemetry: a client span with tracerProvider, in the
// trace of the query's context, and the query's duration with meterProvider. A query's span is named after its
// SQL operation, the statement's first keyword such as SELECT, and its duration is labelled with the same
// db.operation.name; neither carries the query's arguments. Each connection taken from the pool is a span too,
// and the pool's statistics, such as its acquired, idle and maximum connections, are recorded with
// meterProvider.
func WithTelemetry(tracerProvider trace.TracerProvider, meterProvider metric.MeterProvider) Option {
	return func(poolSettings *settings) {
		poolSettings.tracerProvider, poolSettings.meterProvider = tracerProvider, meterProvider
	}
}

// Open connects a pgx pool to url, with options applied, and pings the server within 5 seconds. When url cannot be
// parsed or the server cannot be reached it returns an error and leaves nothing open. Pool settings such as
// pool_max_conns are read from url.
func Open(ctx context.Context, url string, options ...Option) (*pgxpool.Pool, error) {
	var poolSettings settings
	for _, option := range options {
		option(&poolSettings)
	}

	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	if poolSettings.tracerProvider != nil {
		config.ConnConfig.Tracer = otelpgx.NewTracer(
			otelpgx.WithTracerProvider(poolSettings.tracerProvider),
			otelpgx.WithMeterProvider(poolSettings.meterProvider),
			otelpgx.WithMetricOperationName(otelpgx.SQLOperationName),
		)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if poolSettings.meterProvider != nil {
		if err := otelpgx.RecordStats(pool, otelpgx.WithStatsMeterProvider(poolSettings.meterProvider)); err != nil {
			pool.Close()
			return nil, fmt.Errorf("record database pool statistics: %w", err)
		}
	}
	return pool, nil
}
