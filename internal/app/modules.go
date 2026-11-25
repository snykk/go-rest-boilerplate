package app

import (
	"github.com/snykk/go-rest-boilerplate/internal/modules/article"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity"
	"github.com/snykk/go-rest-boilerplate/internal/platform/events"
)

// modules holds every feature module of the application.
type modules struct {
	article  *article.Module
	identity *identity.Module
	comment  *comment.Module
}

// newModules builds every module on the shared resources, one event bus and its own settings. A module that
// needs another one receives that module's API here, and a module that reacts to another's events subscribes
// here, so the dependencies between modules are all visible in one place. It returns an error when a module
// cannot be built.
func newModules(config Config, deps Deps) (modules, error) {
	bus := events.NewBus(deps.Log, deps.Telemetry.TracerProvider)
	users, err := identity.New(identity.Deps{
		Pool:    deps.Pool,
		Log:     deps.Log,
		Mail:    deps.Mail,
		Service: config.Platform.Service,
		Config:  config.Identity,
	})
	if err != nil {
		return modules{}, err
	}
	articles := article.New(article.Deps{
		Pool:   deps.Pool,
		Log:    deps.Log,
		Events: bus,
		Config: config.Article,
	})

	return modules{
		article:  articles,
		identity: users,
		comment: comment.New(comment.Deps{
			Pool:     deps.Pool,
			Log:      deps.Log,
			Articles: articles.API(),
			Authors:  users.API(),
			Config:   config.Comment,
		}),
	}, nil
}
