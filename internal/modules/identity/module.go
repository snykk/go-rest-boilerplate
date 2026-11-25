// Package identity is the identity module: users who register with an e-mail address and a password, verify
// the address with a mailed code, sign in for a short-lived access token and a rotating refresh token, and
// reset or change their password. Its layers are wired together here; the rest of the application sees only
// Module.
package identity

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/mail"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/gateway/postgres"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/transport/rest"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/auth/jwt"
	"github.com/snykk/go-rest-boilerplate/internal/platform/clock"
	"github.com/snykk/go-rest-boilerplate/internal/platform/httpserver"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
	"github.com/snykk/go-rest-boilerplate/internal/platform/password"
	"github.com/snykk/go-rest-boilerplate/internal/platform/random"
)

// Deps are what the identity module takes from the application.
type Deps struct {
	Pool *pgxpool.Pool
	// Log receives the module's security events and the errors of its requests.
	Log *slog.Logger
	// Mail is the queue the module's messages, verification codes and password reset tokens, are sent through.
	Mail mailer.Enqueuer
	// Service is the name of the service, shown in the messages the module sends.
	Service string
	Config  Config
}

// Module serves the identity endpoints and the API other modules call, and provisions the accounts of the
// development seed data.
type Module struct {
	useCases *usecase.UserUsecase
	rest     *rest.Handler
	api      *API
}

// New builds the identity module over deps. It returns an error when the password hasher or the messages
// cannot be prepared: a bcrypt cost out of range, or a template that does not parse.
func New(deps Deps) (*Module, error) {
	hasher, err := password.NewBcrypt(deps.Config.BcryptCost)
	if err != nil {
		return nil, err
	}
	messages, err := mail.NewMail(deps.Mail, deps.Log, mail.Settings{
		Service:  deps.Service,
		CodeTTL:  deps.Config.CodeTTL,
		ResetTTL: deps.Config.ResetTTL,
	})
	if err != nil {
		return nil, err
	}

	useCases := usecase.NewUserUsecase(usecase.UserDeps{
		Users:         postgres.NewUserRepo(deps.Pool),
		Verifications: postgres.NewVerificationRepo(deps.Pool),
		Resets:        postgres.NewResetRepo(deps.Pool),
		Sessions:      postgres.NewSessionRepo(deps.Pool),
		UnitOfWork:    postgres.NewUnitOfWork(deps.Pool),
		Hasher:        hasher,
		Tokens:        jwt.NewIssuer(deps.Config.JWT(), clock.Real{}.Now),
		Random:        random.Generator{},
		Mail:          messages,
		Clock:         clock.Real{},
		IDs:           id.Generator{},
		Settings: usecase.Settings{
			AccessTTL:      deps.Config.AccessTTL,
			RefreshTTL:     deps.Config.RefreshTTL,
			CodeTTL:        deps.Config.CodeTTL,
			ResetTTL:       deps.Config.ResetTTL,
			ReuseGrace:     deps.Config.RefreshReuseGrace,
			Lockout:        deps.Config.Lockout,
			CodeAttempts:   deps.Config.CodeAttempts,
			LoginAttempts:  deps.Config.LoginAttempts,
			ResendCooldown: deps.Config.ResendCooldown,
		},
		Log: deps.Log,
	})
	return &Module{
		useCases: useCases,
		rest:     rest.NewHandler(useCases, deps.Log),
		api:      NewAPI(useCases),
	}, nil
}

// API returns what other modules may call on the identity module.
func (m *Module) API() *API {
	return m.api
}

// RegisterRoutes adds the identity endpoints relative to the groups' prefixes: registration, verification,
// sign-in, refresh, sign-out and password resets to public, and the password change and GET /users/me to
// signedIn, which must authenticate its requests with the access tokens the module issues.
func (m *Module) RegisterRoutes(public, signedIn *httpserver.Router) {
	m.rest.Routes(public, signedIn)
}

// Provision stores a new user with the given roles whose email is already verified. No endpoint grants a role:
// the development seed data creates its editor with Provision, and in production an operator grants the editor
// role to a registered user with SQL. Its errors are those of usecase.UserUsecase.Provision,
// domain.ErrEmailTaken among them.
func (m *Module) Provision(ctx context.Context, input usecase.ProvisionInput) (usecase.UserView, error) {
	return m.useCases.Provision(ctx, input)
}
