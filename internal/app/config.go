package app

import (
	"errors"

	"github.com/snykk/go-rest-boilerplate/internal/modules/article"
	"github.com/snykk/go-rest-boilerplate/internal/modules/comment"
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
	Comment  comment.Config
}

// LoadConfig reads every setting from source, the platform's (see config.FromEnv), the mailer's (see
// mailer.LoadConfig) and each module's, and reports every invalid or missing one in one error. That error also
// names IDENTITY_JWT_SECRET when APP_ENV is production and the secret is the published development value.
func LoadConfig(source config.Source) (Config, error) {
	platform, platformErr := config.FromEnv(source)
	mail, mailErr := mailer.LoadConfig(source)
	articles, articlesErr := article.LoadConfig(source)
	users, usersErr := identity.LoadConfig(source)
	comments, commentsErr := comment.LoadConfig(source)
	secretErr := refuseDevelopmentSecret(source)
	if err := errors.Join(platformErr, mailErr, articlesErr, usersErr, commentsErr, secretErr); err != nil {
		return Config{}, err
	}
	return Config{
		Platform: platform,
		Mail:     mail,
		Article:  articles,
		Identity: users,
		Comment:  comments,
	}, nil
}

// refuseDevelopmentSecret reports IDENTITY_JWT_SECRET when APP_ENV is production and the secret is
// developmentJWTSecret. It reads both settings from source itself, because the platform's and the identity
// module's configurations are empty when they have problems of their own, and this one must be reported with
// them.
func refuseDevelopmentSecret(source config.Source) error {
	reader := config.NewReader(source)
	if reader.String("APP_ENV", "") == "production" &&
		reader.String("IDENTITY_JWT_SECRET", "") == developmentJWTSecret {
		reader.Fail("IDENTITY_JWT_SECRET", "must not be the development value in production")
	}
	return reader.Err()
}
