package app

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
	"github.com/snykk/go-rest-boilerplate/internal/platform/telemetry"
)

// Deps are the shared resources NewHandler builds the application on.
type Deps struct {
	Log *slog.Logger
	// Level is Log's level, which PUT /admin/log-level changes.
	Level     *slog.LevelVar
	Pool      *pgxpool.Pool
	Telemetry *telemetry.Providers
	// Mail takes the email the modules send, such as the identity module's verification codes.
	Mail mailer.Enqueuer
}

// NewHandler returns the application's HTTP handler: the route map of routes.go, served so that a path or
// method no route takes gets a problem. Requests pass through the telemetry (a span and metrics per request),
// request ID, access log, recover, security header, CORS and body limit middleware, in that order. The health
// probes, /health and /ready, get no span and no metrics, and are logged at DEBUG. It returns an error when a
// module cannot be built.
func NewHandler(config Config, deps Deps) (http.Handler, error) {
	featureModules, err := newModules(config, deps)
	if err != nil {
		return nil, err
	}
	router := httpserver.NewRouter()
	routes(router, config, deps, featureModules)

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
	), nil
}
