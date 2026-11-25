package app

import (
	"errors"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity"
	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
)

// developmentJWTSecret is the JWT secret that .env.example and deployment/compose.yml publish for local
// development; anyone who reads the repository can sign access tokens with it.
const developmentJWTSecret = "development-secret-32-bytes-long"

// Config is the configuration of the whole application: the platform's settings, the mailer's and each
// module's own.
type Config struct {
	Platform config.Config
	Mail     mailer.Config
	Article  article.Config
	Identity identity.Config
}

// LoadConfig reads every setting from source, the platform's (see config.FromEnv), the mailer's (see
// mailer.LoadConfig) and each module's, and reports every invalid or missing one in one error. Once they are
// valid, it refuses IDENTITY_JWT_SECRET in production when the secret is the published development value.
func LoadConfig(source config.Source) (Config, error) {
	platform, platformErr := config.FromEnv(source)
	mail, mailErr := mailer.LoadConfig(source)
	articles, articlesErr := article.LoadConfig(source)
	users, usersErr := identity.LoadConfig(source)
	if err := errors.Join(platformErr, mailErr, articlesErr, usersErr); err != nil {
		return Config{}, err
	}
	if platform.Production() && users.JWTSecret == developmentJWTSecret {
		return Config{}, errors.New(
			"invalid configuration: IDENTITY_JWT_SECRET: must not be the development value in production")
	}
	return Config{
		Platform: platform,
		Mail:     mail,
		Article:  articles,
		Identity: users,
	}, nil
}
