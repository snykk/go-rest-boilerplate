// Package article is the article module: drafts addressed by unique slugs, published once. Its layers are
// wired together here; the rest of the application sees only Module.
package article

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/transport/rest"
	"github.com/snykk/go-rest-boilerplate/internal/modules/article/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/clock"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// Deps are what the article module takes from the application.
type Deps struct {
	Pool   *pgxpool.Pool
	Log    *slog.Logger
	Config Config
}

// Module serves the article endpoints and the API other modules call.
type Module struct {
	rest *rest.Handler
	api  *API
}

// New builds the article module over deps.
func New(deps Deps) *Module {
	useCases := usecase.NewArticleUsecase(usecase.ArticleDeps{
		Articles: postgres.NewArticleRepo(deps.Pool),
		Clock:    clock.Real{},
		IDs:      id.Generator{},
		PageSize: usecase.PageSize{Default: deps.Config.DefaultPageSize, Max: deps.Config.MaxPageSize},
	})
	return &Module{rest: rest.NewHandler(useCases, deps.Log), api: NewAPI(useCases)}
}

// API returns what other modules may call on the article module.
func (m *Module) API() *API {
	return m.api
}

// RegisterRoutes adds the article endpoints relative to the groups' prefixes: the public reads of published
// articles to read, and the changes and the reads of drafts to write, so a service can put authentication on
// write alone. Both may be the same group.
func (m *Module) RegisterRoutes(read, write *httpserver.Router) {
	m.rest.Routes(read, write)
}
