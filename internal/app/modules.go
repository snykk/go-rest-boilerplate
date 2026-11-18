package app

import "github.com/snykk/go-rest-boilerplate/internal/modules/article"

// modules holds every feature module of the application.
type modules struct {
	article *article.Module
}

// newModules builds every module on the shared resources and its own settings. A module that needs another
// one receives that module's API here, so the dependencies between modules are all visible in one place.
func newModules(config Config, deps Deps) modules {
	return modules{
		article: article.New(article.Deps{
			Pool:   deps.Pool,
			Log:    deps.Log,
			Config: config.Article,
		}),
	}
}
