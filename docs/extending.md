# Extending the core

The core holds what nearly every service needs and nothing more. This guide names the place where each common
capability plugs in, so it lands in the same spot in every project and the layer rules still hold. Sections
marked *in core* are built and tested; the others are patterns to follow when a feature needs them.

| Need | Where it plugs in | Status |
|---|---|---|
| [Writes that succeed or fail together](#units-of-work) | a `UnitOfWork` port, repositories on `database.Querier` | in core |
| [A module's own settings](#module-settings) | `<module>/config.go`, `internal/app/config.go` | in core |
| [Authentication and roles](#authentication) | an `auth.Authenticator`, route groups in `internal/app/routes.go` | seam in core |
| [Background work](#background-work): consumers, the outbox relay, scheduled jobs, tickers | components of `cmd/server`, `<module>/transport/event/`, `<module>/transport/cron/` | pattern |
| [Events between modules](#events-and-the-outbox) | `<module>/contract/`, an outbox table, a relay | pattern |
| [Cross-cutting concerns for every transport](#cross-cutting-concerns-at-the-use-case-boundary) | a decorator in the module root | pattern |
| [Queries with a shape of their own](#queries-and-read-models) | a query port and a read model | pattern |
| [Caching](#caching) | a decorator over a port | pattern |
| [Calling an external API](#external-apis) | `<module>/gateway/<provider>/` | pattern |
| [Tracing](#tracing-with-opentelemetry) | `internal/platform/telemetry`; spans around use cases and outgoing calls | in core |
| [Metrics](#metrics) | `internal/platform/telemetry`; business metrics in a decorator | in core |
| [Rate limits and idempotency keys](#rate-limits-and-idempotency-keys) | middleware on a route group | pattern |
| [Multi-tenancy](#multi-tenancy) | `auth.Principal`, use case inputs, gateways | pattern |
| [Feature flags](#feature-flags) | a port in `usecase/ports.go` | pattern |
| [Another transport, such as gRPC](#another-transport) | `<module>/transport/grpc/`, a server component | pattern |
| [Moving a module into its own service](#moving-a-module-into-its-own-service) | the consumer's `gateway/<provider>/` | pattern |

## Units of work

*In core.* A use case whose writes go through more than one repository, and must all happen or none, runs them
in one unit of work. The unit of work hands the use case repositories built on one transaction, so the code
shows which calls belong to it. The use case declares the port in its `ports.go`:

```go
// UnitOfWork runs writes that succeed or fail together.
type UnitOfWork interface {
	// Do calls work with repositories that share one transaction. It commits when work returns nil, and rolls
	// back and returns work's error when it does not.
	Do(ctx context.Context, work func(repositories Repositories) error) error
}

// Repositories are the repositories of one unit of work.
type Repositories struct {
	Orders OrderRepository
	Stock  StockRepository
}
```

Inside `work` the use case calls only the repositories it is handed:

```go
err := s.uow.Do(ctx, func(repositories Repositories) error {
	if err := repositories.Orders.Create(ctx, order); err != nil {
		return err
	}
	return repositories.Stock.Reserve(ctx, order.Lines)
})
```

The module's gateway implements the port. Repositories take a `database.Querier`, which a pool and a
transaction both are, so the gateway builds each one on the transaction:

```go
// UnitOfWork runs units of work on the module's tables.
type UnitOfWork struct{ pool *pgxpool.Pool }

func (u *UnitOfWork) Do(ctx context.Context, work func(repositories usecase.Repositories) error) error {
	return pgx.BeginFunc(ctx, u.pool, func(transaction pgx.Tx) error {
		return work(usecase.Repositories{Orders: NewOrderRepo(transaction), Stock: NewStockRepo(transaction)})
	})
}
```

- `Repositories` holds only the module's own repositories, so a unit of work never spans two modules. Another
  module learns of the change through an event (see [Events and the outbox](#events-and-the-outbox)).
- The transaction is never hidden in the context: a call that should join it has to be made through `repositories`.
- Keep units of work short, and never call an external API inside one: the transaction holds its locks until
  the call returns.
- A change to one entity needs no unit of work: `Update(ctx, id, change)` already holds the entity's lock in a
  transaction of its own, or in a savepoint when its repository runs on a transaction.

## Module settings

*In core.* A module reads its own settings in `<module>/config.go`; `article/config.go` is the example.

1. Add a field to the module's `Config` and read it in `LoadConfig` with a `config.Reader`. Name the variable
   after the module (`ARTICLE_MAX_PAGE_SIZE`), and record a rule between settings with `Reader.Fail`.
2. A new module adds its `Config` to `app.Config` and its `LoadConfig` to `app.LoadConfig`, so the
   problems of every module are reported at start-up together.
3. `module.go` hands the values to the parts that need them; use cases receive plain values, such as
   `usecase.PageSize`, never a type from `internal/platform`.
4. Add the variable to `.env.example`, the configuration table in `README.md` and the module card.

A secret, such as an API key, comes from the environment or a secret store, never from a committed file.

## Authentication

*Seam in core.* `internal/platform/auth` holds what every scheme shares: the caller (`Principal`), the
middleware that identifies it (`Authenticate`) and the one that checks its roles (`Require`). The scheme
itself is an `Authenticator`, such as a JWT verifier in `internal/platform/auth/jwt`:

```go
func (v *Verifier) Authenticate(r *http.Request) (auth.Principal, error) {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return auth.Principal{}, auth.ErrUnauthenticated
	}
	claims, err := v.verify(token) // signature, expiry, issuer, audience
	if err != nil {
		return auth.Principal{}, fmt.Errorf("%w: %v", auth.ErrUnauthenticated, err)
	}
	return auth.Principal{ID: claims.Subject, Roles: claims.Roles}, nil
}
```

An error that does not wrap `ErrUnauthenticated`, such as an unreachable key server, answers 500 and is
logged. Build the verifier in `app.open` from its settings, register a closer for anything it runs in the
background, and mount it on route groups in `internal/app/routes.go`:

```go
v1 := router.Group("/v1")
signedIn := v1.Group("", auth.Authenticate(verifier, "Bearer", log))
featureModules.article.RegisterRoutes(v1, signedIn.Group("", auth.Require(auth.Role("editor"))))
featureModules.order.RegisterRoutes(signedIn)
featureModules.report.RegisterRoutes(signedIn.Group("/admin", auth.Require(auth.Role("admin"))))
```

A module with both public and protected endpoints takes one group of each in its `RegisterRoutes`, as
`article` does: anyone reads articles, and only editors change them. The core passes `/v1` for both.

- A handler reads the caller with `auth.PrincipalFrom(r.Context())` and passes what the use case needs in its
  input (`CreateInput{AuthorID: principal.ID, …}`). Use cases never look for the caller in the context, so their
  inputs and tests show who acts.
- Rules about ownership ("only the author edits a draft") belong in the domain or the use case; middleware
  checks only who the caller is and its roles.
- Sign-in, token issuing and refresh are features: they go in a module of their own, such as `identity`.
  Version `v0.2.0` shows one.

## Background work

*Pattern.* Work that does not start with a request runs inside `cmd/server`, as `lifecycle` components beside
the HTTP server. `lifecycle.Run` gives each one a goroutine of its own, cancels them all when the process
stops, and waits for them within `HTTP_SHUTDOWN_TIMEOUT` and one second more, so no goroutine outlives the
process's shutdown.

| Kind | Where | Across replicas |
|---|---|---|
| Queue consumer | `<module>/transport/event/` | the broker's consumer group shares the partitions; a record is acknowledged only after its use case returns |
| Outbox relay | `<module>/transport/event/` | each replica claims a batch (`SELECT … FOR UPDATE SKIP LOCKED`, or a claim lease); a row is published by one replica at a time |
| Scheduled job | `<module>/transport/cron/` | an exclusive job runs only on the replica holding a leader lease, or claims its slot first with a unique row per job and slot |
| Process-local ticker: cache refresh, key reload, buffer flush | `internal/app`, or the platform package it serves | every replica runs its own, since each holds its own state |

A component is a function that runs until its context ends:

```go
// OutboxRelay publishes the outbox every second until ctx ends.
func (a *App) OutboxRelay() lifecycle.ComponentFunc {
	return func(ctx context.Context) error {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				a.relay.PublishBatch(ctx)
			}
		}
	}
}
```

`cmd/server` runs it with the rest: `application.Run(ctx, application.HTTPServer(), application.OutboxRelay(), application.Consumers())`.

- **The rule for a ticker.** If what it changes belongs to the process (a cache, a buffer), every replica
  runs it. If it changes shared data (rows, emails, calls to other services), it is a scheduled job and must
  run once: give it a lease or a slot claim.
- **Consumers and jobs call the use cases**, as handlers do, but no middleware runs around them. Each record or
  run starts its own span, continuing the trace a record carries through `Providers.Propagator`, and puts its
  trace ID in the context with `requestid.With`. Its logs then carry `request_id`, `trace_id` and `span_id`,
  and its SQL queries are child spans; a metric of its own goes where [Metrics](#metrics) shows.
- **Jobs and handlers of records are idempotent**: delivering a record or running a job twice has the effect of
  doing it once.
- **Stopping:** a component returns when its context ends. A consumer finishes the record in hand or leaves it
  unacknowledged for redelivery, and the relay's claims expire for another replica.
- **While they share a process, protect the requests.** Background work gets its own cap on database
  connections, so a backlog cannot empty the pool requests use; consumers process a bounded number of records
  at once; a panic in one record or run is recovered and logged.
- **Splitting is a deployment choice, not new code.** When background load must scale, deploy or fail apart
  from requests (a consumer backlog, jobs that slow the API, long grace periods, other credentials), add a role
  setting such as `SERVER_ROLES=api` and `SERVER_ROLES=events,cron`: the same image then runs as two
  deployments, each starting only its components. Until then, one deployment is simpler.

## Events and the outbox

*Pattern.* Events let a module react to another one without depending on it.

- The publishing module declares its events in `contract/`, with standard library types only:
  `type ArticlePublished struct{ ID string; At time.Time }`.
- The publisher's use case publishes through a port in its `ports.go`, which a gateway implements over an
  in-process bus that the project adds in `internal/platform/events`. The subscribing module's `transport/`
  receives them. The publisher never imports the subscriber.
- An event that must not be lost, such as one bound for a queue, goes into the module's outbox table, such as
  `article_outbox`, in the same unit of work as the change. The relay, a component of `cmd/server` in
  `<module>/transport/event/`, publishes the waiting rows and marks them sent (see
  [Background work](#background-work)). A relay can send a row twice, so subscribers ignore an event ID they
  have handled.

## Cross-cutting concerns at the use-case boundary

*Pattern.* HTTP middleware logs, measures and traces requests, but a job or a gRPC call never passes through
it. When a module gains a second way in, put those concerns around its use cases, once for every transport.

A transport depends on the methods it calls, not on `*usecase.ArticleUsecase` itself, so a decorator with the same
methods fits in between. It lives in the module root, next to `module.go`, which may import platform
packages:

```go
// observed wraps the article use cases with a span per call.
type observed struct {
	next   *usecase.ArticleUsecase
	tracer trace.Tracer
}

func (o observed) Publish(ctx context.Context, id string) (usecase.ArticleView, error) {
	ctx, span := o.tracer.Start(ctx, "article.Publish")
	defer span.End()

	return o.next.Publish(ctx, id)
}
```

- `New` passes the decorator to every transport in place of the service.
- A decorator changes no behaviour: it returns exactly what the use case returns.
- A module with many use cases can move to one handler type per use case, wrapped by generic decorators, as the
  Wild Workouts example does. The core keeps one use case type per aggregate until a module needs more.

## Queries and read models

*Pattern.* `List` loads whole entities through the repository. That is enough while a list shows what one
entity holds. When a screen or a report needs another shape (fields from several tables, totals, a search),
serve it from a read model instead:

- The use case declares a query port in `ports.go` that returns the read model itself, such as
  `ArticleSummaries.Search(ctx, query) ([]ArticleSummary, int, error)`, and returns it as its output. No entity is
  loaded, so no rule runs and nothing can be changed by mistake.
- The gateway implements the port with SQL that selects only what the read model needs. It reads the
  module's own tables, never another module's.
- Transports receive output models such as `usecase.ArticleView`, never entities, so moving a query to a read
  model changes no transport.

## Caching

*Pattern.* A cache wraps a port, so the use cases and their tests do not change. In `gateway/cache/`:

```go
// ArticleRepo is an ArticleRepository that serves ByID from a cache and passes every other call on.
type ArticleRepo struct {
	usecase.ArticleRepository
	cache Cache
}
```

`module.go` wraps the Postgres repository with it. Entries expire after a short time, and the write methods
remove the entries they change. A cache client, such as Redis, is a platform package that `app.open` opens
and closes like the database pool. Measure first: Postgres answers a
lookup by primary key in well under a millisecond.

## External APIs

*Pattern.* The use case declares what it needs as a port, in its own words (`Payments.Charge`), and
`gateway/<provider>/` implements it over the provider's API.

- Every call has a timeout. Only idempotent calls are retried, with backoff.
- The gateway turns the provider's errors into the module's own, as the Postgres gateway does with driver
  errors.
- The base URL and the credentials are module settings.
- Tests: the use case against a mock of the port, the gateway against an `httptest.Server`.

## Tracing with OpenTelemetry

*In core.* `internal/platform/telemetry` builds the tracer provider and the W3C propagator, and
`internal/app` wires them:

- every request continues the caller's `traceparent` and gets a span named after its route;
- every SQL query is a child span named after its operation, without its arguments (`database.WithTelemetry`);
- every log record carries `trace_id` and `span_id`, and a request without `X-Request-ID` uses the trace ID,
  so requests that continue one caller's trace share a request ID and differ by span ID;
- spans are exported over OTLP/HTTP when `OTEL_EXPORTER_OTLP_ENDPOINT` is set, and flushed on shutdown.

What a project adds when it needs it:

- **Spans around use cases**, for work that does not start with an HTTP request: a decorator (see
  [Cross-cutting concerns](#cross-cutting-concerns-at-the-use-case-boundary)) that starts a span from
  `otel.Tracer("<module>")`. `app.New` installs the telemetry as OpenTelemetry's global providers, so
  `otel.Tracer` and `otel.Meter` work anywhere.
- **Outgoing calls:** a gateway's HTTP client uses `otelhttp.NewTransport`, so the trace continues in the
  service it calls.
- **Jobs and consumers:** a job starts a span per item, continuing the trace a message carries in its headers
  through `Providers.Propagator`.

## Metrics

*In core.* The meter provider of `internal/platform/telemetry` pushes metrics over OTLP/HTTP to the
collector at `OTEL_EXPORTER_OTLP_ENDPOINT`, every `METRICS_EXPORT_INTERVAL`. Metrics and traces describe the
process alike: service, version, environment, host name and a `service.instance.id` of its own, so the series
of two replicas never merge; `OTEL_RESOURCE_ATTRIBUTES` adds attributes and may set the instance ID. Without
code in the modules it records:

- `http.server.request.duration` and the body sizes, by method, status and `http.route`, the route pattern the
  `Router` adds, never the raw path. The meter provider drops `server.address` and `server.port`, which come
  from the `Host` header a client sends;
- `db.client.operation.duration`, by SQL operation, and the `pgxpool.*` statistics of the pool;
- the Go runtime's goroutines, memory and garbage collection.

What a project adds when it needs it:

- **Business metrics**, such as articles published, in a decorator around the use cases (see
  [Cross-cutting concerns](#cross-cutting-concerns-at-the-use-case-boundary)), with an instrument from
  `otel.Meter("<module>")`. Attributes have few values: a status or an outcome, never an ID.
- **A metrics backend.** The collector decides where metrics go: point an exporter in
  `deployment/otel-collector.yaml` at Prometheus (remote write), Grafana Mimir or a vendor. The service does not
  change.
- **Scraping instead of pushing**, when a platform only scrapes: the collector's `prometheus` exporter
  serves `/metrics` for it, so the service still only pushes.

## Rate limits and idempotency keys

*Pattern.* Both are middleware on the route groups that need them, in `internal/app/routes.go`.

- **Rate limits** key on the caller's ID, or on `httpserver.ClientIP`, which believes `X-Forwarded-For` only
  from trusted proxies. Replicas share one limit through a shared store such as Redis. Answer 429
  `rate_limited` with a `Retry-After` header.
- **Idempotency keys** make a retried `POST` safe. Store the key, a hash of the request and the response;
  answer a repeated key with the stored response, 409 while the first request with the key is still running,
  and 422 when the key comes back with a different request.

## Multi-tenancy

*Pattern.* The tenant comes from the credentials, never from a value the client sets on its own.

- Add `TenantID` to `auth.Principal`, and set it in the `Authenticator`.
- Handlers pass it in the use case input; repositories filter every query by it, and every unique index
  includes it.
- For a stronger guarantee, Postgres row-level security can enforce the filter: set the tenant at the start
  of each transaction with `set_config('app.tenant_id', $1, true)`.

## Feature flags

*Pattern.* A use case that behaves differently under a flag declares a port for it in `ports.go`
(`Flags.Enabled(ctx, name) bool`), and a gateway reads the flag from its source: a setting, a table or a flag
service. Tests set the flag through the mock, so both behaviours are covered. Delete the flag and its port
when the rollout ends.

## Another transport

*Pattern.* A gRPC API, or any other transport, is one more adapter beside `transport/rest/`.

- `<module>/transport/grpc/` calls the same use cases; the layer rules for `transport/` apply to it.
- The `.proto` files live in `api/proto/`, next to the OpenAPI document.
- The gRPC server is a component that `App` returns beside `HTTPServer`, and `application.Run` runs both. Interceptors
  do the work of the middleware: request IDs, logging, recovery, and authentication that produces the same
  `auth.Principal`.

## Moving a module into its own service

*Pattern.* The module rules keep this move small. A module that others call already exposes only its
`contract/` and `api.go`, and owns its tables.

1. The new service serves the module's use cases over HTTP or gRPC, with the module's tables moved to its own
   database.
2. Each consumer's `gateway/<provider>/` calls that API instead of `api.go`. The consumers' use cases do not
   change.
3. Events that crossed an in-process bus go through the outbox and a queue.
