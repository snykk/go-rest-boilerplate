package app

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// Deps are the shared resources NewHandler builds the application on.
type Deps struct {
	Log  *slog.Logger
	Pool *pgxpool.Pool
}

// NewHandler returns the application's HTTP handler: the route map of routes.go, served so that a path or
// method no route takes gets a problem. Requests pass through the request ID, access log, recover, security
// header, CORS and body limit middleware, in that order. The health probes, /health and /ready, are logged at
// DEBUG.
func NewHandler(config Config, deps Deps) http.Handler {
	router := httpserver.NewRouter()
	routes(router, config, deps, newModules(config, deps))

	probes := []string{"/health", "/ready"}
	return httpserver.Chain(router.Handler(),
		httpserver.RequestID(),
		httpserver.AccessLog(deps.Log, config.Platform.HTTP.TrustedProxies, probes...),
		httpserver.Recover(deps.Log),
		httpserver.SecurityHeaders(config.Platform.Production()),
		httpserver.CORS(config.Platform.HTTP.CORSAllowedOrigins),
		httpserver.BodyLimit(config.Platform.HTTP.BodyLimitBytes),
	)
}
