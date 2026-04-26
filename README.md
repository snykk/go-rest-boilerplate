# go-rest-boilerplate

A starting point for building RESTful APIs in Go using the Gin framework, sqlx for PostgreSQL, and Redis for caching. Implementation follows Clean Architecture principles as described by Uncle Bob.

## Features

### Core
- Clean Architecture layering — `handler → usecase → repository`, no back-imports
- Gin HTTP router with structured access log + request-ID middleware
- PostgreSQL via sqlx, Ristretto + Redis two-tier caching
- Graceful shutdown drains in-flight HTTP, mailer queue, DB, Redis, and tracer in order

### Authentication
- JWT access tokens + refresh tokens (rotation on `/auth/refresh`, revocation on `/auth/logout`)
- HMAC signing-method check + `kind` claim guard against access/refresh confusion
- bcrypt password hashing (configurable cost, default 12)
- OTP-verified registration with brute-force lockout (configurable max attempts)
- Login timing-attack mitigation (dummy bcrypt on user-not-found)

### Observability
- Prometheus metrics: HTTP, cache (hit/miss/error per layer), mailer outcomes, DB pool stats
- OpenTelemetry tracing — HTTP server spans, DB queries (otelsqlx), Redis commands (redisotel), mailer attempts
- Structured request/audit logs (logrus) with X-Request-ID propagation
- `/health` (liveness), `/ready` (DB + Redis probe), `/metrics`, `/swagger`

### Security
- HSTS, CSP, X-Frame-Options, nosniff, Referrer-Policy, Permissions-Policy headers
- CORS origins from env (whitelist), wildcard only in dev
- Per-IP rate limiting on auth endpoints
- 1MB body size limit, configurable HTTP timeouts (Idle, ReadHeader, Read, Write)
- Soft-delete-aware queries; partial unique indexes on `(LOWER(email))` and `(LOWER(username))`

### Testing & DevOps
- Unit tests with mockery-generated mocks
- Integration tests via testcontainers-go (real Postgres + Redis)
- CI: lint (golangci-lint), unit + race, integration with Docker, OpenAPI drift detection, build, multi-arch Docker
- Security scans: govulncheck (deps) + gosec (source) on schedule
- Distroless multi-arch (amd64/arm64) container image
- Async OTP mailer with retry & graceful drain

## Getting Started

### Prerequisites
- Go 1.25+
- PostgreSQL 16+
- Redis 7+
- Docker (only required for integration tests; production runs against managed services)

### Quick start
```bash
git clone https://github.com/snykk/go-rest-boilerplate.git
cd go-rest-boilerplate
cp internal/config/.env.example internal/config/.env
# Edit .env — JWT_SECRET must be at least 32 characters

# Option 1: Docker Compose (full stack)
make docker-up

# Option 2: local dev with hot-reload
make mig-up   # apply migrations
make dev      # air-powered hot reload
```

## Configuration

All configuration is via environment variables loaded from `internal/config/.env`. See [`.env.example`](internal/config/.env.example) for the full list. Highlights:

| Variable | Default | Notes |
| --- | --- | --- |
| `PORT` | 8080 | |
| `ENVIRONMENT` | development | `production` enables HSTS and requires `ALLOWED_ORIGINS` |
| `JWT_SECRET` | _(required, ≥32 chars)_ | HS256 needs 256-bit entropy |
| `JWT_EXPIRED` | 5 | Access token TTL in hours |
| `JWT_REFRESH_EXPIRED` | 7 | Refresh token TTL in days |
| `BCRYPT_COST` | 12 | Range 10–31 |
| `OTP_MAX_ATTEMPTS` | 5 | Lockout threshold per email |
| `MAILER_WORKERS` / `_QUEUE_SIZE` / `_RETRIES` | 2 / 64 / 3 | Async mailer pool |
| `DB_MAX_OPEN_CONNS` / `_IDLE_CONNS` / `_LIFE_MINS` | 25 / 5 / 15 | sqlx pool |
| `OTEL_EXPORTER` | _(empty = disabled)_ | `stdout` for dev, `otlp` for prod |
| `ALLOWED_ORIGINS` | _(empty)_ | Comma-separated; required in production |

## Make targets

```bash
make serve              # run the API directly
make dev                # hot reload via air
make test               # unit tests (mocks only — fast, no Docker)
make test-integration   # integration tests via testcontainers (requires Docker)
make test-cover         # coverage.html
make lint               # golangci-lint
make swag               # regenerate OpenAPI spec from handler annotations
make mig-up / mig-down  # apply / revert migrations (idempotent)
make seed               # seed the database
make docker-up / -down  # full stack via docker-compose
```

## Endpoints

OpenAPI spec is auto-generated from godoc annotations on the handlers. Browse it at:

```
http://localhost:8080/swagger/index.html
```

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| POST | `/api/v1/auth/register` | - | Create account (inactive until OTP verified) |
| POST | `/api/v1/auth/send-otp` | - | Email a 6-digit OTP |
| POST | `/api/v1/auth/verify-otp` | - | Activate account |
| POST | `/api/v1/auth/login` | - | Issue access + refresh token pair |
| POST | `/api/v1/auth/refresh` | - | Rotate refresh, return new pair |
| POST | `/api/v1/auth/logout` | - | Revoke refresh token |
| GET | `/api/v1/users/me` | Bearer | Current user profile |
| GET | `/health` | - | Liveness probe |
| GET | `/ready` | - | Readiness — checks DB + Redis |
| GET | `/metrics` | - | Prometheus scrape endpoint |
| GET | `/swagger/*` | - | OpenAPI UI |

## Folder structure

```
.
├── cmd/
│   ├── api/                # HTTP server entry point + DI wiring
│   ├── migration/          # CLI wrapper around internal/datasources/migration
│   └── seed/
├── deploy/                 # Dockerfile, docker-compose
├── docs/                   # OpenAPI spec (generated by `make swag` — do not edit)
├── internal/
│   ├── apperror/           # Typed error envelope (DomainError + Unwrap)
│   ├── business/
│   │   ├── domains/v1/     # Entity + interfaces (UserDomain, UserRepository, UserUsecase)
│   │   └── usecases/v1/    # Business logic
│   ├── config/             # Viper-backed config + .env.example
│   ├── constants/          # True constants only (sentinel errors, enum values)
│   ├── datasources/
│   │   ├── caches/         # Redis (with OTel hook) + Ristretto
│   │   ├── drivers/        # sqlx wrapped in otelsqlx
│   │   ├── migration/      # Idempotent migration runner (lib + tests)
│   │   ├── records/        # DB row structs
│   │   └── repositories/   # Postgres impl of domain interfaces
│   ├── http/
│   │   ├── auth/           # CurrentUserFromContext helper
│   │   ├── datatransfers/  # Request/Response DTOs
│   │   ├── handlers/v1/    # HTTP handlers + RespondWithError
│   │   ├── middlewares/    # auth, cors, security headers, rate limit, metrics, ...
│   │   └── routes/
│   └── test/
│       ├── mocks/          # mockery-generated test doubles
│       └── testenv/        # testcontainers harness (build-tagged: integration)
├── pkg/
│   ├── audit/              # Auth event JSON-line logger
│   ├── helpers/            # bcrypt, OTP code generator
│   ├── jwt/                # Access + refresh token service
│   ├── logger/             # logrus wrapper, HTTP access log formatter
│   ├── mailer/             # Sync + async OTP mailer (HTML template embedded)
│   ├── observability/      # Prometheus metrics + OpenTelemetry tracing
│   └── validators/         # go-playground/validator + structured FieldError
├── go.mod / go.sum
├── makefile
└── README.md
```

## Testing

```bash
make test               # unit tests, mocks only — runs in seconds
make test-integration   # spins up Postgres + Redis containers; ~30s first run
```

Unit tests live next to the code they test (`*_test.go`). Integration tests are gated behind the `integration` build tag so they're excluded from the default test run; the harness is in `internal/test/testenv/`.

## Observability

Set `OTEL_EXPORTER=stdout` in dev to print spans, or `OTEL_EXPORTER=otlp` in production with `OTEL_EXPORTER_OTLP_ENDPOINT=http://collector:4317`. Spans flow:

```
HTTP server (otelgin)
  ├── DB SELECT/INSERT (otelsqlx) — db.statement, db.system=postgresql
  ├── Redis GET/SET/INCR/EXPIRE (redisotel)
  └── (separate root) mailer.SendOTP per attempt
```

## Contributing

PRs welcome. The CI fails when:
- `go test ./...` fails (race detector enabled)
- `go test -tags=integration ./...` fails (needs Docker, runs in the runner)
- `make swag` regeneration produces a diff — run it locally and commit the result
- govulncheck or gosec flags an issue (security workflow)

## License

MIT — see [LICENSE](LICENSE).
