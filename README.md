# go-rest-boilerplate

A minimal, REST-first starting point for HTTP APIs in Go. Feature modules live under `internal/modules`,
each with Clean Architecture layers inside: a domain, use cases that own their ports, a REST transport and a
Postgres gateway. They are served by `net/http` over shared platform packages. The core ships with one
reference module, `article`, and the seams most services need: units of work, settings per module,
authentication and in-process events. [docs/extending.md](docs/extending.md) shows where everything else plugs
in, such as background jobs, an outbox, caching and rate limits.

This branch, `examples/standard-blog`, is an example built on that core:
[Examples: standard-blog](#examples-standard-blog) says what it adds to the core and how to try it.

## Features

- `net/http` routing with `ServeMux` patterns, graceful shutdown, and health and readiness probes.
- RFC 9457 problem responses with stable error codes and request IDs.
- Middleware: request IDs, structured access logs, panic recovery, security headers, CORS, and body limits.
  Client IPs come from `X-Forwarded-For` only behind trusted proxies.
- PostgreSQL through pgx, with embedded goose migrations that tolerate versions merged out of order.
- Clean Architecture kept honest: entities guard their own state, use cases return plain output models,
  ports belong to the use cases, and a test enforces the import rules.
- Units of work: repositories that run on a pool or on a transaction, so writes through several of them
  commit or roll back together; updates hold the entity's row lock.
- An authentication seam: middleware for any credential scheme, with roles. This example builds JWT sign-in
  into that seam (`internal/platform/auth/jwt` and the `identity` module).
- A process lifecycle: components that run and stop together, and resources closed in reverse order.
- Production logging with `log/slog`, in text or JSON:
  - the service, version and environment on every record;
  - the request, trace and span IDs of the request, and the file and line of the logging call;
  - secrets redacted;
  - a level that operators change while the service runs.
- OpenTelemetry traces and metrics:
  - W3C trace context from callers;
  - a span per request named after its route, and one per SQL query;
  - request durations per route, query durations, connection pool statistics and Go runtime metrics;
  - both pushed over OTLP/HTTP to a collector;
  - request IDs equal to the trace ID;
  - a compose profile with an OpenTelemetry Collector and Jaeger.
- Configuration from environment variables, with an optional `.env` file; each module reads its own settings.
- Validation messages that never repeat the submitted value.
- OpenAPI generated from handler annotations, served with Swagger UI.
- Tests: table-driven unit tests with gomock, integration tests on real Postgres in Docker, and the
  architecture's import rules as a test.
- Tooling: a distroless container image, a compose stack, golangci-lint, and CI with drift checks for the
  generated code.

## Quick start

Requires Docker.

```sh
docker compose -f deployment/compose.yml up --build -d
curl localhost:8080/ready
curl -X POST localhost:8080/v1/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"correct-horse-battery","display_name":"Ada"}'
docker compose -f deployment/compose.yml down -v
```

The stack starts Postgres, applies the migrations once, starts Mailpit, then serves the API on `:8080`. The
verification code of the registration is in Mailpit's inbox at <http://localhost:8025>. The Swagger UI is at
<http://localhost:8080/swagger/index.html>. `docker compose -f deployment/compose.yml --profile seed run --build --rm seed`
adds an editor, a reader, three articles and two comments for local development; it can run again without
adding them twice, and it refuses `APP_ENV=production`.

## Local development

Requires Go 1.26 or newer and a PostgreSQL server.

```sh
cp .env.example .env      # then point DATABASE_URL at your database
make migrate-up
make seed     # optional, development only: an editor, a reader, three articles and two comments
make run
```

`make help` lists every target. [AGENTS.md](AGENTS.md) holds the rules for changing the code: architecture,
conventions, and the definition of done.

## Examples: standard-blog

`examples/standard-blog` shows what a typical single-team service looks like once it grows from the core: several
modules in one database, users who sign in, modules that call each other without breaking the import rules, and
a module that reacts to another one through an event. Read its history as a learning path, and copy the
patterns you need.

### What it adds to `master`

- **Mail.** `internal/platform/mailer` sends plain-text and HTML email over SMTP, through a bounded in-memory
  queue that `cmd/server` runs beside the HTTP server. The compose stack runs Mailpit, which keeps every
  message instead of delivering it.
- **Sign-in.** `internal/platform/auth/jwt` signs and verifies HS256 access tokens, and `internal/platform/password`
  and `internal/platform/random` hash passwords and draw codes. The `identity` module builds on them: register,
  verify the email with a mailed code, sign in, refresh with rotating tokens, sign out, reset or change a password,
  lock an account after repeated failures, and log security events. Users are `reader`s or `editor`s.
- **Roles on the article routes.** Reading published articles stays public; creating, changing, publishing and
  deleting them, and reading drafts, need a signed-in editor. `internal/app/routes.go` puts
  `auth.Authenticate` and `auth.Require` on the groups, and the article module's code does not change; only its
  endpoint annotations now declare the bearer scheme and the 401 and 403 answers in the OpenAPI document.
- **A second module that calls two others.** `comment` lets a signed-in user comment on a published article. It
  reaches `article` and `identity` only through their `contract/` packages, wrapped in its own
  `gateway/article` and `gateway/identity`.
- **An event against the direction of a dependency.** `article` publishes `ArticleDeleted` and `comment` removes
  the article's comments, so `article` never knows `comment`.
- **Seed data.** `cmd/seed` creates an editor, a reader, three articles and two comments for local development.

### Run it

```sh
make docker-up        # Postgres, the migrations, Mailpit and the API on :8080
docker compose -f deployment/compose.yml --profile seed run --build --rm seed
```

The second command is the seed data; `--rm` leaves no stopped container behind for `make docker-down` to miss.
Mailpit's inbox, with the emails the API sends, is at <http://localhost:8025>. `make docker-down` stops the stack
and deletes its database volume. With your own PostgreSQL, as in [Local development](#local-development),
`make seed` creates the same data; run it after `make migrate-up`.

The seed accounts are `editor@example.com` with the password `editor-password` and `reader@example.com` with
`reader-password`. Their emails and passwords come from the `SEED_` settings, and running the seed again adds
nothing twice.

### Flows to try

The commands below use `sed` to pick values out of the JSON answers; `jq` does the same where you have it.

```sh
api=localhost:8080/v1
json='Content-Type: application/json'
field() { sed -E "s/.*\"$1\":\"([^\"]+)\".*/\1/"; }
login() {
  curl -s -X POST $api/auth/login -H "$json" -d "{\"email\":\"$1\",\"password\":\"$2\"}" | field access_token
}
```

**An editor writes and publishes an article, and anyone reads it.**

```sh
editor=$(login editor@example.com editor-password)
article=$(curl -s -X POST $api/articles -H "$json" -H "Authorization: Bearer $editor" \
  -d '{"slug":"my-first-post","title":"My first post","body":"Written by an editor."}' | field id)
curl -s -o /dev/null -w '%{http_code}\n' $api/articles/$article                 # 404: a draft is not public
curl -s $api/articles/drafts -H "Authorization: Bearer $editor"                  # the editor's drafts
curl -s -X POST $api/articles/$article/publish -H "Authorization: Bearer $editor"
curl -s $api/articles/$article                                                    # now anyone can read it
```

**A reader comments; only an editor writes articles.**

```sh
reader=$(login reader@example.com reader-password)
curl -s -X POST $api/comments -H "$json" -H "Authorization: Bearer $reader" \
  -d "{\"article_id\":\"$article\",\"body\":\"Nice post.\"}"
curl -s "$api/comments?article_id=$article"                                       # anyone lists the comments
curl -s -X POST $api/articles -H "$json" -H "Authorization: Bearer $reader" \
  -d '{"slug":"nope","title":"Nope","body":"A reader may not."}'                  # 403 forbidden
```

**Deleting the article removes its comments.**

```sh
curl -s -o /dev/null -w '%{http_code}\n' -X DELETE $api/articles/$article -H "Authorization: Bearer $editor"   # 204
curl -s "$api/comments?article_id=$article"                                       # 404 not_found
```

**A new account registers, verifies the email, signs in, refreshes and signs out.**

```sh
curl -s -X POST $api/auth/register -H "$json" \
  -d '{"email":"ada@example.com","password":"correct-horse-battery","display_name":"Ada"}'
sleep 1                                                                          # the mail goes out in the background
code=$(curl -s -G localhost:8025/api/v1/search --data-urlencode 'query=to:ada@example.com' \
  | sed -E 's/.*verify your email address: ([0-9]{6}).*/\1/')                      # or read it at http://localhost:8025
curl -s -X POST $api/auth/verification -H "$json" -d "{\"email\":\"ada@example.com\",\"code\":\"$code\"}"   # 204
pair=$(curl -s -X POST $api/auth/login -H "$json" -d '{"email":"ada@example.com","password":"correct-horse-battery"}')
curl -s $api/users/me -H "Authorization: Bearer $(echo "$pair" | field access_token)"
pair=$(curl -s -X POST $api/auth/refresh -H "$json" -d "{\"refresh_token\":\"$(echo "$pair" | field refresh_token)\"}")
curl -s -X POST $api/auth/logout -H "$json" -d "{\"refresh_token\":\"$(echo "$pair" | field refresh_token)\"}"   # 204
```

The password reset is the same shape: `POST /v1/auth/password/forgot` mails a token that
`POST /v1/auth/password/reset` takes with the new password.

### Learning path

One commit per step, each of them building and passing its tests;
`git log --reverse --oneline --no-merges master..HEAD` lists them in this order.

| Step | Commit | What to read |
|---|---|---|
| 1 | `feat(mailer): add an SMTP mailer with a background queue and Mailpit` | `internal/platform/mailer`: a platform package with settings of its own, and a queue run as a lifecycle component |
| 2 | `feat(jwt): add a JWT issuer and an authenticator that verifies it` | `internal/platform/auth/jwt`: a scheme that plugs into the authentication seam |
| 3 | `feat(identity): add the domain of users, passwords, sessions and secrets` | `internal/modules/identity/domain`: entities that keep their state unexported |
| 4 | `feat(identity): add use cases for sign-in, verification and resets` | `internal/modules/identity/usecase`: ports, a unit of work and security events |
| 5 | `feat(password): add bcrypt password hashing with a constant-cost dummy` | `internal/platform/password`: a technical capability that satisfies a port without a gateway |
| 6 | `feat(random): add uniform verification codes and URL-safe tokens` | `internal/platform/random` |
| 7 | `feat(identity): add the Postgres and mail gateways and the tables` | `identity/gateway` and the migration: conditional updates, and mail from templates |
| 8 | `feat(identity): add the REST endpoints and wire sign-in into the app` | `identity/transport/rest` and `internal/app`: where authentication meets the route groups |
| 9 | `feat(comment): add comments on published articles through module APIs` | `comment/gateway/article` and `comment/gateway/identity`: calls between modules through `contract/` |
| 10 | `feat(comment): remove a deleted article's comments on ArticleDeleted` | `comment/transport/event`: an event against the direction of a dependency |
| 11 | `feat(seed): add seed data that runs twice` | `cmd/seed`: development data through the application's own handler |
| 12 | `docs: describe the standard example and run CI on example branches` | this section, and CI on `examples/**` branches with a compose smoke test |

## Configuration

Variables come from the environment and, when present, from `./.env`; the environment wins. An empty value
counts as unset.

| Variable | Default | Notes |
|---|---|---|
| `APP_ENV` | `development` | `development` or `production` |
| `SERVICE_NAME` | `go-rest-boilerplate` | The service in log records and traces |
| `HTTP_ADDR` | `:8080` | |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` | Go duration syntax |
| `HTTP_READ_TIMEOUT` | `15s` | |
| `HTTP_WRITE_TIMEOUT` | `15s` | |
| `HTTP_IDLE_TIMEOUT` | `60s` | |
| `HTTP_SHUTDOWN_TIMEOUT` | `10s` | How long requests in flight may run after SIGTERM; the components get a second more to stop, then the resources as long again to close, so a container's stop grace period must exceed twice this (`deployment/compose.yml` sets 25s) |
| `HTTP_BODY_LIMIT_BYTES` | `1048576` | |
| `HTTP_TRUSTED_PROXIES` | none | Comma-separated CIDRs of reverse proxies whose `X-Forwarded-For` is trusted |
| `HTTP_CORS_ALLOWED_ORIGINS` | `*` in development | Comma-separated; required in production |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `text` in development, `json` in production | `text` or `json` |
| `LOG_SOURCE` | `true` | Adds `source`, the file and line of each logging call |
| `SWAGGER_ENABLED` | `true` in development, `false` in production | |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | none | OTLP/HTTP base URL of the collector traces and metrics are pushed to, such as `http://collector:4318`; the other `OTEL_EXPORTER_OTLP_*` variables of the process environment also apply, and `OTEL_RESOURCE_ATTRIBUTES` adds resource attributes |
| `TRACING_SAMPLE_RATIO` | `1` | Share of new traces recorded, 0 to 1; a trace the caller sampled is always recorded; metrics are never sampled |
| `METRICS_EXPORT_INTERVAL` | `60s` | How often metrics are pushed |
| `ADMIN_TOKEN` | none | Bearer token of the operator endpoints under `/admin`, at least 32 characters; unset, they do not exist |
| `DATABASE_URL` | required | pgx connection string; pool settings through URL parameters such as `pool_max_conns` |
| `MAIL_SMTP_ADDR` | `localhost:1025` | Host and port of the SMTP server. The default is Mailpit from `deployment/compose.yml`, published on the host at `localhost:1025` (the `api` container uses `mailpit:1025`); its inbox is at `http://localhost:8025` |
| `MAIL_FROM` | `no-reply@example.com` | The sender of every message, with an optional display name such as `Blog <blog@example.com>` |
| `MAIL_SMTP_USERNAME` | none | Authenticates with PLAIN when set; sent only over STARTTLS or to the local host |
| `MAIL_SMTP_PASSWORD` | none | The password for `MAIL_SMTP_USERNAME` |
| `MAIL_SMTP_STARTTLS` | `false` | `true` upgrades the connection with STARTTLS and fails when the server does not offer it |
| `MAIL_TIMEOUT` | `10s` | The limit of one delivery as a whole |
| `MAIL_QUEUE_SIZE` | `100` | Messages held in memory before new ones are dropped |
| `ARTICLE_DEFAULT_PAGE_SIZE` | `20` | Articles in a list that asks for no `limit` |
| `ARTICLE_MAX_PAGE_SIZE` | `100` | The most articles a list returns; not below `ARTICLE_DEFAULT_PAGE_SIZE` |
| `IDENTITY_JWT_SECRET` | required | Signs and verifies access tokens (HS256); at least 32 bytes, random, kept secret; the development value of `.env.example` is refused in production |
| `IDENTITY_JWT_ISSUER` | `go-rest-boilerplate` | The `iss` claim of an access token |
| `IDENTITY_JWT_AUDIENCE` | `go-rest-boilerplate` | The `aud` claim of an access token |
| `IDENTITY_ACCESS_TTL` | `15m` | Lifetime of an access token, which stays valid until then after a sign-out; at least `1m` |
| `IDENTITY_REFRESH_TTL` | `720h` | Lifetime of a refresh token; at least `1m` |
| `IDENTITY_REFRESH_REUSE_GRACE` | `10s` | How long after a refresh the spent token may come again without being taken as stolen |
| `IDENTITY_CODE_TTL` | `10m` | Lifetime of an email verification code; at least `1m` |
| `IDENTITY_CODE_ATTEMPTS` | `5` | Wrong guesses that end a verification code |
| `IDENTITY_RESET_TTL` | `30m` | Lifetime of a password reset token; at least `1m` |
| `IDENTITY_RESEND_COOLDOWN` | `60s` | How long after a verification code or a reset token was issued a request for another one answers as usual but issues and mails nothing |
| `IDENTITY_BCRYPT_COST` | `12` | From 10 to 14 |
| `IDENTITY_LOGIN_ATTEMPTS` | `5` | Failed sign-ins in a row that lock an account |
| `IDENTITY_LOCKOUT` | `15m` | How long a locked account stays locked |
| `COMMENT_DEFAULT_PAGE_SIZE` | `20` | Comments in a list that asks for no `limit` |
| `COMMENT_MAX_PAGE_SIZE` | `100` | The most comments a list returns; not below `COMMENT_DEFAULT_PAGE_SIZE` |
| `SEED_EDITOR_EMAIL` | `editor@example.com` | Local development only: the email of the editor `make seed` creates |
| `SEED_EDITOR_PASSWORD` | `editor-password` | Local development only: the editor's password, 8 to 72 bytes |
| `SEED_READER_EMAIL` | `reader@example.com` | Local development only: the email of the reader `make seed` creates |
| `SEED_READER_PASSWORD` | `reader-password` | Local development only: the reader's password, 8 to 72 bytes |

A setting whose name starts with a module's name, such as `ARTICLE_`, belongs to that module and is read by
its `config.go`; the `SEED_` settings belong to `cmd/seed`; and the `MAIL_` settings are read by
`mailer.LoadConfig` in `internal/platform/mailer`, not by `internal/platform/config`. Every invalid or missing
setting is reported at start-up in one error.

`make seed` and the `SEED_` settings are for local development only: `cmd/seed` refuses to run with
`APP_ENV=production`, and the production image, the default target of `deployment/Dockerfile`, does not contain it.

## API

| Method | Path | Who | Success |
|---|---|---|---|
| GET | `/health` | anyone | 200 while the process serves |
| GET | `/ready` | anyone | 200 when the database answers, 503 otherwise |
| POST | `/v1/auth/register` | anyone | 201, an unverified user; the code goes by email |
| POST | `/v1/auth/verification/resend` | anyone | 202 |
| POST | `/v1/auth/verification` | anyone | 204 |
| POST | `/v1/auth/login` | anyone | 200, an access token and a refresh token |
| POST | `/v1/auth/refresh` | anyone | 200, the next pair |
| POST | `/v1/auth/logout` | anyone | 204 |
| POST | `/v1/auth/password/forgot` | anyone | 202; the reset token goes by email |
| POST | `/v1/auth/password/reset` | anyone | 204 |
| POST | `/v1/auth/password/change` | signed in | 200, a new pair |
| GET | `/v1/users/me` | signed in | 200 |
| POST | `/v1/articles` | editor | 201 with `Location` |
| GET | `/v1/articles?limit=&offset=` | anyone | 200, published articles, newest first |
| GET | `/v1/articles/{id}` | anyone | 200 for a published article, 404 for a draft |
| GET | `/v1/articles/drafts?limit=&offset=` | editor | 200, drafts, newest first |
| GET | `/v1/articles/drafts/{id}` | editor | 200 for a draft |
| PATCH | `/v1/articles/{id}` | editor | 200 |
| POST | `/v1/articles/{id}/publish` | editor | 200 |
| DELETE | `/v1/articles/{id}` | editor | 204 |
| POST | `/v1/comments` | signed in | 201, a comment on a published article |
| GET | `/v1/comments?article_id=&author_id=&limit=&offset=` | anyone | 200, the comments on a published article, newest first |
| DELETE | `/v1/comments/{id}` | the comment's author, or an editor | 204 |

A signed-in request carries `Authorization: Bearer <access_token>`; without a valid one it answers 401
`unauthenticated`, and without the role it needs, 403 `forbidden`. Everyone who registers is a reader; an
editor is an account an operator provisions, since no endpoint grants a role.

### Editors in production

Seed data is for development only. In production a user registers and verifies their email like anyone else,
then an operator grants the editor role with one SQL statement on the `users` table of the `identity` module,
whose `roles` column is `text[]`:

```sql
UPDATE users SET roles = array_append(roles, 'editor') WHERE email = 'ada@example.com' AND NOT 'editor' = ANY(roles);
```

Emails are stored in lower case. The statement can run twice without adding the role twice, and no password
hash is ever written by hand. The role takes effect at the user's next sign-in or refresh; an access token
already issued keeps the roles it was issued with until it expires (`IDENTITY_ACCESS_TTL`).

### Errors

Errors are `application/problem+json`:

```json
{
  "type": "about:blank",
  "title": "Conflict",
  "status": 409,
  "detail": "slug is already used by another article",
  "code": "slug_taken",
  "request_id": "0193f0a2-6c1e-7b3d-9a52-3f0f6a1d2c44"
}
```

A `422 validation_failed` problem adds `"errors": [{"field": "title", "rule": "required", "message": "is
required"}]`. Every response carries `X-Request-ID`. The error codes of each endpoint are listed in the
module cards, [internal/modules/article/README.md](internal/modules/article/README.md),
[internal/modules/identity/README.md](internal/modules/identity/README.md) and
[internal/modules/comment/README.md](internal/modules/comment/README.md), and in the OpenAPI document at
`/swagger/doc.json`.

## Observability

- **Logs.**
  - Every record carries `service`, `version` and `env`. Inside a request it also carries `request_id`,
    `trace_id` and `span_id`, and with `LOG_SOURCE` it carries `source`, such as
    `internal/modules/article/transport/rest/errors.go:36`.
  - The value of a key that names a secret, such as `password`, `token` or `authorization`, is written as
    `[REDACTED]`.
  - Probes (`/health`, `/ready`) are logged at `DEBUG`.
- **Traces.**
  - A request continues the caller's `traceparent`. Its span is named after its route, such as
    `GET /v1/articles/{id}`.
  - Each SQL query is a child span named after its operation, without its arguments.
  - Without `OTEL_EXPORTER_OTLP_ENDPOINT`, the IDs still exist but no span is pushed.
  - A request without `X-Request-ID` gets its trace ID as its request ID, so one ID finds it in the logs
    and in the traces. Requests that continue one caller's trace therefore share a request ID; their span
    IDs tell them apart.
- **Metrics**, pushed to the same collector every `METRICS_EXPORT_INTERVAL`:
  - `http.server.request.duration`, by method, status and `http.route`, never by raw path or `Host` header;
  - `db.client.operation.duration`, by SQL operation;
  - `pgxpool.*`, such as acquired, idle and maximum connections and the time spent waiting for one;
  - `go.*`: goroutines, memory and garbage collection.
- **Resource.** Spans and metrics name the process with `service.name`, `service.version`,
  `deployment.environment.name`, `host.name` and a `service.instance.id` of its own, so the series of two
  replicas never merge. `OTEL_RESOURCE_ATTRIBUTES` adds attributes, such as the pod's, and may set the
  instance ID; a malformed value stops the start-up. Errors of the telemetry itself, such as a failed export,
  are logged at `WARN`.
- **Collector and Jaeger locally:**

  ```sh
  OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318 docker compose -f deployment/compose.yml --profile telemetry up --build
  ```

  Traces are at <http://localhost:16686>. The collector prints the metrics to its log, and
  [deployment/otel-collector.yaml](deployment/otel-collector.yaml) is where a real metrics backend, such as
  Prometheus or Grafana Mimir, would be configured.
- **Log level at runtime**, when `ADMIN_TOKEN` is set:

  ```sh
  curl -X PUT localhost:8080/admin/log-level -H "Authorization: Bearer $ADMIN_TOKEN" -d '{"level":"debug"}'
  ```

- **Build info.** The image's `VERSION` and `COMMIT` build arguments set the version on logs and traces,
  for example `docker build --build-arg VERSION=v1.0.0 --build-arg COMMIT=$(git rev-parse HEAD) …`.

## Project structure

```
cmd/server, cmd/migrate     the API and the migrations
cmd/seed                    development seed data: two accounts, three articles and two comments
internal/app                composition root: process lifecycle, modules, route map, middleware
internal/modules/article    the reference module: domain, usecase, transport/rest, gateway/postgres,
                            gateway/eventbus, contract, api.go
internal/modules/identity   users and sign-in: domain, usecase, transport/rest, gateway/postgres,
                            gateway/mail, contract, api.go
internal/modules/comment    comments: domain, usecase, transport/rest, transport/event, gateway/postgres,
                            gateway/article, gateway/identity
internal/platform/*         config, logger, telemetry, buildinfo, requestid, id, random, clock, validate,
                            database, migrate, httpserver, auth (and auth/jwt), password, mailer, events,
                            lifecycle
internal/archtest           the import rules as a test
internal/testenv            Postgres for integration tests
migrations/                 SQL migrations, embedded into the migrate binary
api/openapi                 generated OpenAPI document
deployment/                 Dockerfile (with a seed target), compose.yml (with Mailpit) and the OpenTelemetry
                            Collector configuration
docs/extending.md           where each common capability plugs in
```

## Adding a module

Follow [internal/modules/README.md](internal/modules/README.md): the module anatomy, the steps, and the
rules for ports and for calls between modules.

## Extending

[docs/extending.md](docs/extending.md) names the place for each capability a growing service adds: units of
work, module settings, authentication, background jobs, events and the outbox, caching, external APIs,
OpenTelemetry, rate limits, idempotency keys, multi-tenancy, feature flags and gRPC.

## Testing

```sh
make test               # unit tests and the architecture test
make test-integration   # also the tests against Postgres in Docker
make lint
```

- **Unit tests** are table-driven, one test function per unit, with gomock mocks generated by `make generate`.
- **Integration tests** carry the build tag `integration`. Each test binary starts one Postgres container,
  and each test gets a database of its own in it.
- **The architecture test** (`internal/archtest`) fails when a package imports across a forbidden boundary.

On Windows with Docker Desktop, run the integration tests with `TESTCONTAINERS_RYUK_DISABLED=true`, and keep
running the packages one at a time (`make test-integration` does).

## Versions

| Version | Release | Structure |
|---|---|---|
| [`v0.1.0`](https://github.com/snykk/go-rest-boilerplate/tree/v0.1.0) | go-rest-boilerplate initial release | Gin, layers versioned under `v1`, sqlx, JWT and OTP activation |
| [`v0.2.0`](https://github.com/snykk/go-rest-boilerplate/tree/v0.2.0) | go-rest-boilerplate bounded contexts release | `users` and `auth` bounded contexts, refresh tokens, observability |
| [`v0.3.0`](https://github.com/snykk/go-rest-boilerplate/tree/v0.3.0) | go-rest-boilerplate modular core release | A minimal modular monolith on `net/http`: one reference module, platform packages, architecture test |

`git checkout v0.2.0` shows an earlier structure in full. [CHANGELOG.md](CHANGELOG.md) records what each
release changed, and [RELEASING.md](RELEASING.md) describes how a change becomes a release.

## Contributing

[CONTRIBUTING.md](CONTRIBUTING.md) describes how to propose a change, and [SECURITY.md](SECURITY.md) how to
report a vulnerability privately. Everyone taking part follows the [Code of Conduct](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE)
