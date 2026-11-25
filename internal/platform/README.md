# Platform

Technical plumbing that every module may use and none owns: configuration, logging, IDs, time, validation,
the database pool and units of work, the migrator, the HTTP server with its middleware, authentication, tracing
and metrics, build information, and the process lifecycle.

## What belongs here

Ask: if this package moved to another repository, would it carry a business word such as `Article`, `Order`
or `Customer`? If it would, it is not platform code. Infrastructure that only one module uses stays in that
module's `gateway/` or `transport/`.

## Rules

- A platform package imports the standard library, third-party packages and other platform packages, never
  `internal/modules/` or `internal/app`. `internal/archtest` enforces this.
- Platform packages hold no global state that callers configure; the composition root in `internal/app`
  builds and passes what is needed.
- Every exported identifier documents its contract, as for the rest of the code (see
  [AGENTS.md](../../AGENTS.md)).

## Packages

| Package | Provides |
|---|---|
| `config` | `LoadSource`, `FromEnv`, `LoadDatabase` and `Reader`: settings from the environment and an optional `./.env`, every problem reported at once; a module reads its own settings with a `Reader` |
| `logger` | `New`, `Options`: a `log/slog` logger in `text` or `json` with the given attributes on every record, `request_id`, `trace_id` and `span_id` from the context, the caller's `source`, and secrets redacted; its level may be a `slog.LevelVar` |
| `telemetry` | `New`, `Config`, `Providers`: the OpenTelemetry tracer provider (sampling at a ratio), meter provider (with Go runtime metrics) and W3C propagator, pushing over OTLP/HTTP when a collector endpoint is set; `Shutdown` flushes both. The resource adds the host and a `service.instance.id`; a view drops `server.address` and `server.port` from the `http.server.*` metrics; `ErrorHandler` logs OpenTelemetry's own errors |
| `buildinfo` | `Get`, `Info`: the version and commit of the binary, from `-ldflags` or what the Go toolchain recorded |
| `requestid` | `Header`, `With`, `From`, `Valid`: the request ID in a context, and the check for incoming IDs |
| `id` | `New`, `Generator`: UUIDv7 identifiers |
| `clock` | `Real`: the current time in UTC, in whole microseconds |
| `validate` | `Struct`, `Errors`, `FieldError`: struct validation whose messages never repeat the submitted value |
| `database` | `Open`: a pgx pool, pinged before use, with `WithTelemetry` for a span and a duration per query and the pool's statistics; `Querier`: a pool or a transaction, which gateways take so the same repository runs alone or in a unit of work |
| `migrate` | `New`: a goose provider over embedded SQL migrations, out-of-order versions allowed; apart from `database` so the server binary does not carry goose |
| `httpserver` | `Run`, `Serve`, `Config`: graceful serving; `Chain` and the middleware `Instrument` (a span and metrics per request), `RequestID`, `AccessLog`, `Recover`, `SecurityHeaders`, `CORS`, `BodyLimit`; `ClientIP`; `Problem`, `WriteProblem`, `ValidationProblem`, `WriteJSON`, `DecodeJSON`; `Health`, `Ready`, `Check`; `LogLevel`, the operator handler for a `slog.LevelVar`; `Router` with `Group`, `Handle` and `Handler`, which names each request's span, and labels its metrics, after its route and answers unknown paths and methods with problems |
| `auth` | `Authenticate`, `Require`, `Role`: middleware that identifies the caller through an `Authenticator` (401 `unauthenticated`) and checks what it may do (403 `forbidden`); `Principal`, `WithPrincipal`, `PrincipalFrom`: the caller in a context. `StaticToken`: a bearer token for operator endpoints. No user credential scheme is built in: a JWT or session verifier implements `Authenticator` |
| `events` | `NewBus`, `Subscribe`, `Publish`: an in-process event bus. Handlers of an event's exact type run in the publisher's goroutine, each in a span of its own; a handler's error or panic is logged and stops nothing. Delivery is at most once |
| `lifecycle` | `Component`, `ComponentFunc`, `Run`: components that run together and stop together within a timeout; `Closers`: resources closed in reverse order |
