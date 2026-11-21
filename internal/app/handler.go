package app

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/telemetry"
)

// Deps are the shared resources NewHandler builds the application on.
type Deps struct {
	Log       *slog.Logger
	Pool      *pgxpool.Pool
	Telemetry *telemetry.Providers
}

// NewHandler returns the application's HTTP handler: the route map of routes.go, served so that a path or
// method no route takes gets a problem. Requests pass through the telemetry (a span and metrics per request),
// request ID, access log, recover, security header, CORS and body limit middleware, in that order. The health
// probes, /health and /ready, get no span and no metrics, and are logged at DEBUG.
func NewHandler(config Config, deps Deps) http.Handler {
	router := httpserver.NewRouter()
	routes(router, config, deps, newModules(config, deps))

	probes := []string{"/health", "/ready"}
	return httpserver.Chain(router.Handler(),
		httpserver.Instrument(deps.Telemetry.TracerProvider, deps.Telemetry.MeterProvider, deps.Telemetry.Propagator,
			probes...),
		httpserver.RequestID(),
		httpserver.AccessLog(deps.Log, config.Platform.HTTP.TrustedProxies, probes...),
		httpserver.Recover(deps.Log),
		httpserver.SecurityHeaders(config.Platform.Production()),
		httpserver.CORS(config.Platform.HTTP.CORSAllowedOrigins),
		httpserver.BodyLimit(config.Platform.HTTP.BodyLimitBytes),
	)
}
