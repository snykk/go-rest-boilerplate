// Package comment is the comment module: signed-in users comment on published articles, anyone reads the
// comments, and a comment's author or an editor deletes it. It asks the article module whether an article is
// published and the identity module for an author's display name, each through that module's API. Its layers
// are wired together here; the rest of the application sees only Module.
package comment

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/gateway/article"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/gateway/identity"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/transport/rest"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/clock"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
)

// Deps are what the comment module takes from the application.
type Deps struct {
	Pool *pgxpool.Pool
	Log  *slog.Logger
	// Articles is the article module's API, which tells whether an article is published.
	Articles article.API
	// Authors is the identity module's API, which gives the display name of a comment's author.
	Authors identity.API
	Config  Config
}

// Module serves the comment endpoints.
type Module struct {
	rest *rest.Handler
}

// New builds the comment module over deps.
func New(deps Deps) *Module {
	useCases := usecase.NewCommentUsecase(usecase.CommentDeps{
		Comments: postgres.NewCommentRepo(deps.Pool),
		Articles: article.NewArticles(deps.Articles),
		Authors:  identity.NewAuthors(deps.Authors),
		Clock:    clock.Real{},
		IDs:      id.Generator{},
		PageSize: usecase.PageSize{Default: deps.Config.DefaultPageSize, Max: deps.Config.MaxPageSize},
	})
	return &Module{rest: rest.NewHandler(useCases, deps.Log)}
}

// RegisterRoutes adds the comment endpoints relative to the groups' prefixes: the list to public, and adding
// and deleting a comment to signedIn, which must authenticate its requests with the access tokens of the
// identity module.
func (m *Module) RegisterRoutes(public, signedIn *httpserver.Router) {
	m.rest.Routes(public, signedIn)
}
