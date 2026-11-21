package app

import (
	"net/http"
	"time"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/snykk/go-rest-boilerplate/api/openapi" // registers the document served at /swagger/doc.json
	"github.com/snykk/go-rest-boilerplate/internal/platform/auth"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
)

// routes is the application's route map: the platform routes at the root, the operator routes under /admin
// when an admin token is configured, and every module mounted on the /v1 group. A group that needs its own
// middleware, such as authentication, is created here.
func routes(router *httpserver.Router, config Config, deps Deps, featureModules modules) {
	router.Handle(http.MethodGet, "/health", httpserver.Health())
	router.Handle(http.MethodGet, "/ready", httpserver.Ready(deps.Log, 2*time.Second, httpserver.Check{
		Name:  "database",
		Probe: deps.Pool.Ping,
	}))
	if config.Platform.Swagger {
		router.Handle(http.MethodGet, "/swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))
	}

	if token := config.Platform.Admin.Token; token != "" {
		operator := auth.Principal{ID: "operator", Roles: []string{"admin"}}
		admin := router.Group("/admin", auth.Authenticate(auth.StaticToken(token, operator), "Bearer", deps.Log))
		admin.Handle(http.MethodGet, "/log-level", httpserver.LogLevel(deps.Level, deps.Log))
		admin.Handle(http.MethodPut, "/log-level", httpserver.LogLevel(deps.Level, deps.Log))
	}

	v1 := router.Group("/v1")
	featureModules.article.RegisterRoutes(v1, v1)
}
