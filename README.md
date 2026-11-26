# go-rest-boilerplate

A minimal, REST-first starting point for HTTP APIs in Go. Feature modules live under `internal/modules`,
each with Clean Architecture layers inside: a domain, use cases that own their ports, a REST transport and a
Postgres gateway. They are served by `net/http` over shared platform packages. The core ships with one
reference module, `article`, and the seams most services need: units of work, settings per module,
authentication and in-process events. [docs/extending.md](docs/extending.md) shows where everything else plugs
in, such as background jobs, an outbox, caching and rate limits.

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
- An authentication seam: middleware for any credential scheme, with roles. No scheme is built in.
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
curl -X POST localhost:8080/v1/articles -H 'Content-Type: application/json' \
  -d '{"slug":"hello-world","title":"Hello, world","body":"The first article."}'
docker compose -f deployment/compose.yml down -v
```

The stack starts Postgres, applies the migrations once, then serves the API on `:8080`. The Swagger UI is at
<http://localhost:8080/swagger/index.html>.

## Local development

Requires Go 1.26 or newer and a PostgreSQL server.

```sh
cp .env.example .env      # then point DATABASE_URL at your database
make migrate-up
make run
```

`make help` lists every target. [AGENTS.md](AGENTS.md) holds the rules for changing the code: architecture,
conventions, and the definition of done.

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
| `ARTICLE_DEFAULT_PAGE_SIZE` | `20` | Articles in a list that asks for no `limit` |
| `ARTICLE_MAX_PAGE_SIZE` | `100` | The most articles a list returns; not below `ARTICLE_DEFAULT_PAGE_SIZE` |

A setting whose name starts with a module's name, such as `ARTICLE_`, belongs to that module and is read by
its `config.go`. Every invalid or missing setting is reported at start-up in one error.

## API

| Method | Path | Success |
|---|---|---|
| GET | `/health` | 200 while the process serves |
| GET | `/ready` | 200 when the database answers, 503 otherwise |
| POST | `/v1/articles` | 201 with `Location` |
| GET | `/v1/articles?limit=&offset=` | 200, published articles, newest first |
| GET | `/v1/articles/{id}` | 200 for a published article, 404 for a draft |
| GET | `/v1/articles/drafts?limit=&offset=` | 200, drafts, newest first |
| GET | `/v1/articles/drafts/{id}` | 200 for a draft |
| PATCH | `/v1/articles/{id}` | 200 |
| POST | `/v1/articles/{id}/publish` | 200 |
| DELETE | `/v1/articles/{id}` | 204 |

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
module card, [internal/modules/article/README.md](internal/modules/article/README.md), and in the OpenAPI
document at `/swagger/doc.json`.

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
cmd/server, cmd/migrate     the two binaries
internal/app                composition root: process lifecycle, modules, route map, middleware
internal/modules/article    the reference module: domain, usecase, transport/rest, gateway/postgres,
                            gateway/eventbus, contract, api.go
internal/platform/*         config, logger, telemetry, buildinfo, requestid, id, clock, validate, database,
                            migrate, httpserver, auth, events, lifecycle
internal/archtest           the import rules as a test
internal/testenv            Postgres for integration tests
migrations/                 SQL migrations, embedded into the migrate binary
api/openapi                 generated OpenAPI document
deployment/                 Dockerfile, compose.yml and the OpenTelemetry Collector configuration
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

`git checkout v0.2.0` shows an earlier structure in full.

## License

[MIT](LICENSE)
