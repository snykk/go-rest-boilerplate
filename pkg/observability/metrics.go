// Package observability exposes Prometheus metric helpers that can be
// called from any layer without creating an import cycle between the
// business packages and the HTTP middleware that registers collectors.
//
// Collectors are registered here once at init(); callers use the
// Observe* functions below to record events. The HTTP layer separately
// registers its own request-scoped collectors.
package observability

import (
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	cacheOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Cache operations by layer (ristretto|redis), operation, and result (hit|miss|error|ok).",
		},
		[]string{"layer", "op", "result"},
	)

	mailerOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mailer_operations_total",
			Help: "OTP mailer outcomes: sent, failed, queue_full.",
		},
		[]string{"result"},
	)

	dbPoolOpen = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "db_pool_open_connections",
			Help: "Open Postgres connections (idle + in use)",
		},
		func() float64 { return float64(currentDBStats().OpenConnections) },
	)
	dbPoolInUse = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "db_pool_in_use_connections",
			Help: "Postgres connections currently in use",
		},
		func() float64 { return float64(currentDBStats().InUse) },
	)
	dbPoolWait = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "db_pool_wait_count_total",
			Help: "Cumulative connections that waited for a free slot",
		},
		func() float64 { return float64(currentDBStats().WaitCount) },
	)
)

func init() {
	prometheus.MustRegister(cacheOpsTotal, mailerOpsTotal, dbPoolOpen, dbPoolInUse, dbPoolWait)
}

// dbStatsProvider is wired at startup so the GaugeFunc callbacks can
// read live sql.DBStats without importing the server package.
var dbStatsProvider func() *sqlx.DB

type dbStatsSnapshot struct {
	OpenConnections int
	InUse           int
	WaitCount       int64
}

func currentDBStats() dbStatsSnapshot {
	if dbStatsProvider == nil {
		return dbStatsSnapshot{}
	}
	db := dbStatsProvider()
	if db == nil {
		return dbStatsSnapshot{}
	}
	s := db.Stats()
	return dbStatsSnapshot{
		OpenConnections: s.OpenConnections,
		InUse:           s.InUse,
		WaitCount:       s.WaitCount,
	}
}

// RegisterDBStatsProvider must be called once at startup with the live
// *sqlx.DB handle so pool-stats gauges can read it on each scrape.
func RegisterDBStatsProvider(provider func() *sqlx.DB) {
	dbStatsProvider = provider
}

// ObserveCacheOp records one cache operation outcome.
//
//	layer:  "ristretto" | "redis"
//	op:     "get" | "set" | "del"
//	result: "hit" | "miss" | "ok" | "error"
func ObserveCacheOp(layer, op, result string) {
	cacheOpsTotal.WithLabelValues(layer, op, result).Inc()
}

// ObserveMailerOp records one mailer outcome.
//
//	result: "sent" | "failed" | "queue_full"
func ObserveMailerOp(result string) {
	mailerOpsTotal.WithLabelValues(result).Inc()
}
