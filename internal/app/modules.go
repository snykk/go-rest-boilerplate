package app

import (
	"github.com/snykk/go-rest-boilerplate/internal/modules/article"
	"github.com/snykk/go-rest-boilerplate/internal/platform/events"
)

// modules holds every feature module of the application.
type modules struct {
	article *article.Module
}

// newModules builds every module on the shared resources, one event bus and its own settings. A module that
// needs another one receives that module's API here, and a module that reacts to another's events subscribes
// here, so the dependencies between modules are all visible in one place.
func newModules(config Config, deps Deps) modules {
	bus := events.NewBus(deps.Log, deps.Telemetry.TracerProvider)
	return modules{
		article: article.New(article.Deps{
			Pool:   deps.Pool,
			Log:    deps.Log,
			Events: bus,
			Config: config.Article,
		}),
	}
}
