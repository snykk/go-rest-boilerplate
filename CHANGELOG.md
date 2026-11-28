# Changelog

All notable changes to go-rest-boilerplate, newest first. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/): each release groups its changes under Added,
Changed, Removed and Fixed, in that order. Versions follow [Semantic Versioning](https://semver.org/) with a
`v` prefix.

Each version is an annotated git tag on `master`. Merging a release's pull request tags its merge commit, and
the release workflow publishes the version's section below as the GitHub Release notes;
[RELEASING.md](RELEASING.md) describes the flow.

## Versioning guide

Before v1.0.0:

- **MINOR** (`v0.4.0`): new capabilities, and any change to the layout or the rules that a project built on
  the boilerplate follows.
- **PATCH** (`v0.3.1`): fixes only, in the code, the tests or the documents.

## Unreleased

Changes on `master` that are not in a release yet.

### Added

- **Secret settings that never print**: `config.Secret` holds a setting such as a password or a token, and
  `fmt`, `log/slog` and JSON never show its value: they show `[REDACTED]` for the secret itself or an exported
  field that holds it, and at most a pointer through an unexported field. Only `Reveal` returns the value.
- **JWT access tokens**: `internal/platform/auth/jwt` signs HS256 tokens for a subject and its roles and
  verifies them as an `auth.Authenticator`, with the algorithm pinned and the expiry, subject, issuer and
  audience required; a secret shorter than 32 bytes or a missing issuer or audience stops the process at
  start-up. `docs/extending.md` shows how to mount it.

### Changed

- **`ADMIN_TOKEN` and `DATABASE_URL` are secrets**: `config.Admin.Token` and `config.Database.URL` are
  `config.Secret`, so code that sets them builds them with `config.NewSecret` and code that reads them calls
  `Reveal`.

## [v0.3.0] - 2026-11-27

The third release rebuilds go-rest-boilerplate as a minimal modular monolith on the standard library's
`net/http`: one reference module, `article`, in domain, use case, transport and gateway layers whose imports an
architecture test enforces, over small platform packages. Configuration comes from the environment with an
optional `.env`, errors are RFC 9457 problem documents, JSON bodies are decoded strictly, logs go through
`log/slog` with request and trace IDs and redaction, and Postgres is reached through pgx with units of work
and goose migrations embedded in a `migrate` command. Around it come OpenTelemetry traces and metrics pushed
over OTLP, an authentication seam with operator endpoints behind a static token, an in-process event bus, one
lifecycle that shuts the process down gracefully, a client IP read only through trusted proxies, OpenAPI
generated from annotations with Swagger UI, integration tests on testcontainers, a distroless image, a compose
stack, CI, a release workflow, guides for contributors and security reports, and an AGENTS.md that states the
rules for people and coding agents, on Go 1.26.

### Added

- **Modules with enforced layers**: each feature lives in `internal/modules/<name>` as domain, use case,
  transport and gateway packages, reaches another module only through its `contract/` and `api.go`, and
  `internal/archtest` fails `make test` on an import that breaks the rules of AGENTS.md.
  ([c0226f4](https://github.com/snykk/go-rest-boilerplate/commit/c0226f4e5eb856c4bc1ec08cafdf535b6cd13592))
- **The `article` module as the reference to copy**: drafts with unique slugs that are published once, use
  cases that return output models and declare their own ports, a Postgres repository with one file per
  method, and REST endpoints whose reads and changes sit on separate route groups, with drafts served only to
  the change group.
  ([9b22ec7](https://github.com/snykk/go-rest-boilerplate/commit/9b22ec7d336e060be07b664fc54bcee03055ea92),
  [ca0f069](https://github.com/snykk/go-rest-boilerplate/commit/ca0f0695f89e003ce019b05bd5b52b8d59a08e98),
  [649e55b](https://github.com/snykk/go-rest-boilerplate/commit/649e55b5928ec1daeab93d53cf2be568c7e6a644),
  [76fbe0d](https://github.com/snykk/go-rest-boilerplate/commit/76fbe0ded8304fe4613cde367413523901627aa0))
- **Configuration from the environment**: an optional `./.env` never overrides the environment, and every
  invalid or missing setting is reported at start-up in one error.
  ([6277694](https://github.com/snykk/go-rest-boilerplate/commit/62776945dee9c05c7c5eea88f17341a646584488))
- **Platform packages**: request IDs, a `log/slog` logger that adds the request and trace IDs and redacts
  secret-looking keys, build info stamped at build time, a UTC clock and UUIDv7 identifiers.
  ([c55692f](https://github.com/snykk/go-rest-boilerplate/commit/c55692f650ce5ae5482365eea9a00cd7c5c1f539))
- **Validation that never echoes a submitted value**, so a password or a token cannot reach a problem document
  or a log record.
  ([276e268](https://github.com/snykk/go-rest-boilerplate/commit/276e268c09adbbcb6e4ca74a835946e0d5f5e6d5))
- **An HTTP platform**: RFC 9457 problem documents with stable codes; strict JSON decoding that refuses unknown
  fields, trailing data and oversized bodies; request ID, access log, panic recovery, security headers, CORS
  and body limit middleware; a router over `ServeMux` that answers unknown paths and methods with problems;
  health and readiness handlers; graceful shutdown.
  ([051f4b1](https://github.com/snykk/go-rest-boilerplate/commit/051f4b14771fc40384840fac0a1c1110c4545e30),
  [3e69dce](https://github.com/snykk/go-rest-boilerplate/commit/3e69dce9e2475e0ce19ef02f4e3d16449999e0fc),
  [3acf85a](https://github.com/snykk/go-rest-boilerplate/commit/3acf85afa2bcba41c527c81f2c718eaa7a91091b))
- **Postgres through pgx**: a pool pinged before use, repositories on a `Querier` so that one repository runs
  alone or in a unit of work, goose migrations embedded in `cmd/migrate`, and a test harness with one
  container per test binary and a database per test.
  ([9e6a3dd](https://github.com/snykk/go-rest-boilerplate/commit/9e6a3dd5e13f213fb14e0a8cac0717ee80a53893))
- **One composition root and lifecycle**: `internal/app` builds the modules, the routes and the middleware,
  runs components such as the HTTP server and stops them within the shutdown timeout; `cmd/server` and
  `cmd/migrate`, and the OpenAPI document generated from annotations, served with Swagger UI.
  ([dcd2cc9](https://github.com/snykk/go-rest-boilerplate/commit/dcd2cc9ab49477fba5542ee1584b09c47d654c2c))
- **OpenTelemetry traces and metrics**, pushed over OTLP/HTTP when a collector is configured: spans and
  metrics per route and per query, and a compose profile with a Collector and Jaeger.
  ([8455da1](https://github.com/snykk/go-rest-boilerplate/commit/8455da186e4a0911ebfd211e9a7d80b06eaafa3a))
- **An authentication seam**: a principal in the request context, middleware that authenticates and checks
  roles on route groups, and operator endpoints under `/admin` behind `ADMIN_TOKEN`, starting with the
  runtime log level.
  ([0ee1cb1](https://github.com/snykk/go-rest-boilerplate/commit/0ee1cb17e8b61cd9dd89389f8acdd42d11078457))
- **An in-process event bus**, synchronous and at most once, with a span per handler; `article` announces
  `ArticleDeleted`, so other modules can react without `article` depending on them.
  ([1d8879e](https://github.com/snykk/go-rest-boilerplate/commit/1d8879e05d83d3ecda36493e333e6895d50c76d4))
- **Build and CI**: make targets, a distroless nonroot image with the version stamped in, a compose stack that
  migrates once before the API starts, and CI with lint, unit and integration tests under the race detector,
  and checks that the mocks and the OpenAPI document are generated.
  ([52cf8a1](https://github.com/snykk/go-rest-boilerplate/commit/52cf8a10cfc8da1c91cd6b8f48201cc9d8c2588f))
- **Guides for people and coding agents**: AGENTS.md with the rules, a README at each decision point, and
  `docs/extending.md`, which places the capabilities the core does not ship, such as background jobs, an
  outbox or caching.
  ([53ab1f2](https://github.com/snykk/go-rest-boilerplate/commit/53ab1f2e429d4c35ce70e37af2b70fd798cec668))
- **A release flow**: changes reach `master` through release branches and pull requests, with CI on both,
  and merging a release's pull request tags it and publishes its GitHub Release from this changelog
  ([RELEASING.md](RELEASING.md)).
  ([93e98a0](https://github.com/snykk/go-rest-boilerplate/commit/93e98a05756da2c911bb26e4a0a8cd477797475b),
  [c81418a](https://github.com/snykk/go-rest-boilerplate/commit/c81418a92606e7d0506e37b792be3fff50d5f0c4),
  [9079751](https://github.com/snykk/go-rest-boilerplate/commit/907975160d99203788b1e28bb7184c9e87fc7258))
- **Community files**: a code of conduct, a contributing guide, a security policy that takes reports through
  GitHub's private vulnerability reporting, issue forms for bugs and feature requests, and a pull request
  template.
  ([42d4757](https://github.com/snykk/go-rest-boilerplate/commit/42d47571d10cb939021cac70bb9dcab021de6b73),
  [c41864e](https://github.com/snykk/go-rest-boilerplate/commit/c41864eced42b0abee8a0cb1691aa301aa7ce407),
  [9652943](https://github.com/snykk/go-rest-boilerplate/commit/9652943cf35696b5e8f413a964f7f1dd543b0f85))

### Changed

- **The stack moved to the standard library and OpenTelemetry**: Gin gave way to `net/http`, zap to
  `log/slog`, sqlx to pgx, and Prometheus scraping to OpenTelemetry metrics pushed over OTLP. The Go version
  is 1.26.
  ([ba9956b](https://github.com/snykk/go-rest-boilerplate/commit/ba9956b6cd2449d37419de25d3ed27bd479ee927))

### Removed

- **The `users` and `auth` bounded contexts, Redis, the Ristretto cache and the seed command** leave the core,
  and `git checkout v0.2.0` shows the earlier structure.
  ([ba9956b](https://github.com/snykk/go-rest-boilerplate/commit/ba9956b6cd2449d37419de25d3ed27bd479ee927))

### Fixed

- **A forged `X-Forwarded-For` no longer changes the client IP**: the header is read only through the
  configured trusted proxies.
  ([3e69dce](https://github.com/snykk/go-rest-boilerplate/commit/3e69dce9e2475e0ce19ef02f4e3d16449999e0fc))
- **A submitted value never appears in a validation error or a log record.**
  ([276e268](https://github.com/snykk/go-rest-boilerplate/commit/276e268c09adbbcb6e4ca74a835946e0d5f5e6d5))
- **The image and compose start from environment variables alone**; a `.env` file is optional.
  ([6277694](https://github.com/snykk/go-rest-boilerplate/commit/62776945dee9c05c7c5eea88f17341a646584488))
- **Migrations run in version order**, from goose files with timestamped versions, instead of being ordered
  as strings.
  ([9e6a3dd](https://github.com/snykk/go-rest-boilerplate/commit/9e6a3dd5e13f213fb14e0a8cac0717ee80a53893))

## [v0.2.0] - 2026-05-08

The second release splits the business layer into users and auth bounded contexts over a self-contained
domain package, with one file per use case and request and response types at the use case boundary. Auth
now brings refresh tokens with rotation and revocation, OTP activation with lockout, login lockout, password
change, forgot, and reset, and an audit log. Around it come zap structured logging with request and trace
IDs, OpenTelemetry tracing of HTTP, Postgres, Redis, and the mailer, Prometheus metrics, health and readiness
probes, security headers, rate limiting, Swagger generated from handler annotations, an idempotent migration
runner, integration tests on real Postgres and Redis through testcontainers, a distroless multi-arch image,
and CI with golangci-lint, the race detector, integration tests, OpenAPI drift, govulncheck, and gosec, on
Go 1.25.

Its known issues are listed in its
[GitHub Release](https://github.com/snykk/go-rest-boilerplate/releases/tag/v0.2.0).

## [v0.1.0] - 2023-04-26

The first release of go-rest-boilerplate, a starting point for REST APIs in Go: a Gin server laid out in
Clean Architecture layers, with handlers, use cases, and repositories versioned under v1, over PostgreSQL
through sqlx. It covers registration, login with JWT access tokens, account activation by a six-digit OTP
sent through Gmail SMTP and kept in Redis, and the current user's profile behind the auth middleware, cached
in Ristretto, with logrus logging, CORS, migrations, seeding, a hand-written Swagger file, a Dockerfile with
docker-compose, and unit tests on testify mocks, on Go 1.19.

Its known issues are listed in its
[GitHub Release](https://github.com/snykk/go-rest-boilerplate/releases/tag/v0.1.0).

[v0.3.0]: https://github.com/snykk/go-rest-boilerplate/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/snykk/go-rest-boilerplate/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/snykk/go-rest-boilerplate/releases/tag/v0.1.0
