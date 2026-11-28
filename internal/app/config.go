package app

import (
	"errors"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article"
	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
)

// Config is the configuration of the whole application: the platform's settings and each module's own.
type Config struct {
	Platform config.Config
	Article  article.Config
}

// LoadConfig reads every setting from source, the platform's (see config.FromEnv) and each module's, and reports
// every invalid or missing one in one error.
func LoadConfig(source config.Source) (Config, error) {
	platform, platformErr := config.FromEnv(source)
	articles, articlesErr := article.LoadConfig(source)
	if err := errors.Join(platformErr, articlesErr); err != nil {
		return Config{}, err
	}
	return Config{
		Platform: platform,
		Article:  articles,
	}, nil
}
