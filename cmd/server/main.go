// Command server serves the REST API until it receives SIGINT or SIGTERM, then shuts down gracefully.
// Configuration comes from environment variables and an optional ./.env file.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/snykk/go-rest-boilerplate/internal/app"
	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
)

// main runs the API and exits with status 1 when it cannot start or serve.
//
//	@title			Go REST Boilerplate API
//	@version		1.0
//	@description	Minimal REST core: platform packages and the article reference module.
//	@license.name	MIT
//	@BasePath		/v1
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				An access token from POST /auth/login, sent as "Bearer <access_token>".
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	context.AfterFunc(ctx, stop) // after the first signal, a second one ends the process at once

	source, err := config.LoadSource()
	if err != nil {
		return err
	}
	appConfig, err := app.LoadConfig(source)
	if err != nil {
		return err
	}

	application, err := app.New(ctx, appConfig)
	if err != nil {
		return err
	}
	return application.Run(ctx, application.HTTPServer(), application.MailQueue())
}
