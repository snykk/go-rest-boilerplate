# Changelog

All notable changes to go-rest-boilerplate, newest first. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/): each release groups its changes under Added,
Changed, Removed and Fixed, in that order. Versions follow [Semantic Versioning](https://semver.org/) with a
`v` prefix.

## Versioning guide

Before v1.0.0:

- **MINOR** (`v0.4.0`): new capabilities, and any change to the layout or the rules that a project built on
  the boilerplate follows.
- **PATCH** (`v0.3.1`): fixes only, in the code, the tests or the documents.

## Unreleased

### Added

- **Modules with enforced layers**: each feature lives in `internal/modules/<name>` as domain, use case,
  transport and gateway packages, reaches another module only through its `contract/` and `api.go`, and
  `internal/archtest` fails `make test` on an import that breaks the rules of AGENTS.md.
- **The `article` module as the reference to copy**: drafts with unique slugs that are published once, use
  cases that return output models and declare their own ports, a Postgres repository with one file per
  method, and REST endpoints whose reads and changes sit on separate route groups, with drafts served only to
  the change group.
- **Configuration from the environment**: an optional `./.env` never overrides the environment, and every
  invalid or missing setting is reported at start-up in one error.
- **Platform packages**: request IDs, a `log/slog` logger that adds the request and trace IDs and redacts
  secret-looking keys, build info stamped at build time, a UTC clock and UUIDv7 identifiers.
- **Validation that never echoes a submitted value**, so a password or a token cannot reach a problem document
  or a log record.
- **An HTTP platform**: RFC 9457 problem documents with stable codes; strict JSON decoding that refuses unknown
  fields, trailing data and oversized bodies; request ID, access log, panic recovery, security headers, CORS
  and body limit middleware; a router over `ServeMux` that answers unknown paths and methods with problems;
  health and readiness handlers; graceful shutdown.
- **Postgres through pgx**: a pool pinged before use, repositories on a `Querier` so that one repository runs
  alone or in a unit of work, goose migrations embedded in `cmd/migrate`, and a test harness with one
  container per test binary and a database per test.
- **One composition root and lifecycle**: `internal/app` builds the modules, the routes and the middleware,
  runs components such as the HTTP server and stops them within the shutdown timeout; `cmd/server` and
  `cmd/migrate`, and the OpenAPI document generated from annotations, served with Swagger UI.
- **OpenTelemetry traces and metrics**, pushed over OTLP/HTTP when a collector is configured: spans and
  metrics per route and per query, and a compose profile with a Collector and Jaeger.
- **An authentication seam**: a principal in the request context, middleware that authenticates and checks
  roles on route groups, and operator endpoints under `/admin` behind `ADMIN_TOKEN`, starting with the
  runtime log level.
- **An in-process event bus**, synchronous and at most once, with a span per handler; `article` announces
  `ArticleDeleted`, so other modules can react without `article` depending on them.
- **Build and CI**: make targets, a distroless nonroot image with the version stamped in, a compose stack that
  migrates once before the API starts, and CI with lint, unit and integration tests under the race detector,
  and checks that the mocks and the OpenAPI document are generated.
- **Guides for people and coding agents**: AGENTS.md with the rules, a README at each decision point, and
  `docs/extending.md`, which places the capabilities the core does not ship, such as background jobs, an
  outbox or caching.

### Changed

- **The stack moved to the standard library and OpenTelemetry**: Gin gave way to `net/http`, zap to
  `log/slog`, sqlx to pgx, and Prometheus scraping to OpenTelemetry metrics pushed over OTLP. The Go version
  is 1.26.

### Removed

- **The `users` and `auth` bounded contexts, Redis, the Ristretto cache and the seed command** leave the core,
  and `git checkout v0.2.0` shows the earlier structure.

### Fixed

- **A forged `X-Forwarded-For` no longer changes the client IP**: the header is read only through the
  configured trusted proxies.
- **A submitted value never appears in a validation error or a log record.**
- **The image and compose start from environment variables alone**; a `.env` file is optional.
- **Migrations run in version order**, from goose files with timestamped versions, instead of being ordered
  as strings.

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

[v0.2.0]: https://github.com/snykk/go-rest-boilerplate/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/snykk/go-rest-boilerplate/releases/tag/v0.1.0
