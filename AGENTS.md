# AGENTS.md

Rules for everyone who changes this repository, people and coding agents alike. Read this file before you
change code. The READMEs it links add the details for their folders.

## Project map

| Path | Contents |
|---|---|
| `cmd/server` | The server binary: signals and configuration, then `app.New` and `Run` with the HTTP server component. It holds the general OpenAPI annotations. |
| `cmd/migrate` | The migration binary: `up`, `down`, `status`, `create <name>`. |
| `internal/app` | The composition root, the only package that knows every concrete type: `app.go` (the process lifecycle: `New`, `HTTPServer`, `Run`), `config.go` (the settings of the platform and of every module), `handler.go` (router and middleware), `modules.go` (every module and what it is built from) and `routes.go` (the route map). |
| `internal/modules/<name>` | One feature module each, with its own layers. See [internal/modules/README.md](internal/modules/README.md). |
| `internal/platform/*` | Technical plumbing shared by every module, with no business knowledge. See [internal/platform/README.md](internal/platform/README.md). |
| `internal/archtest` | The import rules below, as a test. |
| `internal/testenv` | The integration-test harness: one Postgres container per test binary, a database per test. |
| `migrations/` | The SQL migrations of every module, embedded into the `migrate` binary. See [migrations/README.md](migrations/README.md). |
| `api/openapi` | The generated OpenAPI document. Never edit it by hand. |
| `docs/` | `extending.md`: where each common capability plugs in, such as jobs, events, caching or tracing. |
| `deployment/` | `Dockerfile` (distroless, nonroot), `compose.yml` (Postgres, the migrations once, the API; a `telemetry` profile with an OpenTelemetry Collector and Jaeger) and `otel-collector.yaml`. |

The reference module is `article`; its card is [internal/modules/article/README.md](internal/modules/article/README.md).
A capability the core lacks, such as background jobs, events or caching, goes where
[docs/extending.md](docs/extending.md) places it.

## Architecture rules

| Folder | May import | Must not import |
|---|---|---|
| `<module>/domain/` | standard library | everything else |
| `<module>/contract/` | standard library | everything else |
| `<module>/usecase/` | its own `domain/` and `contract/`, standard library | `transport/`, `gateway/`, `internal/platform/`, `internal/app` |
| `<module>/transport/` | its own `usecase/` and `domain/`, `internal/platform/*`, another module's `contract/` | any `gateway/` |
| `<module>/gateway/` | its own `usecase/` (port types) and `domain/`, `internal/platform/*`, another module's `contract/` | any `transport/` |
| `<module>/` root: `module.go`, `config.go`, `api.go` | its own subpackages, `internal/platform/*` | other modules |
| `internal/platform/` | standard library, third-party packages, other platform packages | `internal/modules/`, `internal/app` |
| `internal/app`, `cmd/` | everything | — |

- Another module is reached only through its `contract/` package, and only from `transport/` or `gateway/`.
  The consumer wraps the provider's `api.go` behind a small interface in its own `gateway/<provider>/`.
- **`gateway/`** holds only adapters that cross the module's boundary: external systems (the database, a cache, a
  broker, mail, an HTTP API), another module (its ACL over that module's `api.go`), and the events the module
  publishes. A technical capability (time, identifiers, password hashing, randomness, token signing) comes from a
  platform package whose type satisfies the use case's port directly, as `clock.Real` and `id.Generator` do in
  `article/module.go`; it needs no gateway.
- Module dependencies never form a cycle; the reverse direction uses events.
- Modules never share tables, and no foreign key points from one module's table to another's.

`internal/archtest` enforces the table: `make test` fails on a forbidden import. The layers are the rings of
Clean Architecture; [internal/modules/README.md](internal/modules/README.md#the-layers-as-clean-architecture)
maps them, with the places where this layout departs from the textbook on purpose.

## Commands

| Command | What it does |
|---|---|
| `make run` | Runs the API; it reads `./.env` when present. |
| `make test` | Runs the unit and architecture tests. |
| `make test-integration` | Runs every test, including those against Postgres in Docker, one package at a time. |
| `make lint` | Runs golangci-lint. |
| `make generate` | Regenerates the gomock mocks from the `//go:generate` lines. |
| `make generate-check` | Fails when the committed mocks differ from freshly generated ones. |
| `make openapi` | Regenerates `api/openapi` from the swag annotations. |
| `make openapi-check` | Fails when the committed OpenAPI files differ from freshly generated ones. |
| `make migrate-up`, `migrate-down`, `migrate-status` | Applies, rolls back or lists migrations (needs `DATABASE_URL`). |
| `make migrate-create name=<module>_<action>` | Creates an empty migration in `migrations/`. |
| `make docker-up`, `make docker-down` | Starts or stops the compose stack (`down` deletes its volume). |
| `make tidy` | Tidies `go.mod` and `go.sum`. |

Tools: Go 1.26 or newer, Docker, and golangci-lint v2.15.0
(`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0`). `mockgen` and `swag` are
pinned in `go.mod` and run through `go tool`.

**Windows with Docker Desktop:** run the integration tests with `TESTCONTAINERS_RYUK_DISABLED=true`, because
the testcontainers reaper does not run there; `testenv.Main` removes the containers instead. Keep running the
packages one at a time (`make test-integration` uses `-p 1`): when several test binaries start at once,
testcontainers can fail to find Docker Desktop's engine.

## Adding a module

1. Create `internal/modules/<name>/` with:
   - `domain/`: entities, rules and sentinel errors, standard library only;
   - `usecase/`: `ports.go` with every outbound need and its `//go:generate` line, `<aggregate>.go` (the use
     case type, `ArticleUsecase`, with its deps and constructor), and one file
     per use case;
   - `transport/rest/`: `usecases.go` (`UseCases`, the interface the handlers call, with its `//go:generate` line),
     `handler.go`, `routes.go` (the module's endpoints), one file per endpoint with its swag annotations
     and request type, `response.go` and `errors.go`;
   - `gateway/postgres/`: `<aggregate>_repo.go`, `<aggregate>_row.go` and one file per repository method;
   - `module.go`: `Deps`, `New` and `RegisterRoutes`, which takes the route groups the endpoints go on, such
     as one for reads and one for changes in `article`;
   - `config.go`, when the module has settings: `Config` and `LoadConfig(source config.Source)`;
   - `README.md`: the module card.
2. Run `make migrate-create name=<name>_<action>` and write the SQL.
3. Build the module in `internal/app/modules.go`, add its settings to `internal/app/config.go`, and mount it
   in `internal/app/routes.go`: on the `/v1` group, or on a group with its own middleware.
4. Write the tests: use cases and handlers with mocks, the gateway against Postgres through `testenv`.
5. Run `make generate openapi`, then go through the definition of done.

The steps in detail are in [internal/modules/README.md](internal/modules/README.md).

## Conventions

### HTTP

- Errors are RFC 9457 problem documents (`application/problem+json`) with the extension members `code`,
  `request_id` and, for validation failures, `errors`. That includes a path no route matches (404
  `not_found`) and a method the route does not take (405 `method_not_allowed`), through `Router.Handler`.
- Problem codes are stable, lower snake case words: `not_found`, `method_not_allowed`, `slug_taken`,
  `validation_failed`, `malformed_json`, `body_too_large`, `invalid_query`, `internal`. A module maps its
  domain errors to codes in its `transport/rest/errors.go`.
- An error with no mapping answers 500 `internal`; its cause is logged and never returned.
- A submitted value never appears in a problem or a log record.
- A transport's request and response types are unexported. Each one names its OpenAPI schema with swag's
  `// @name` after its closing brace (`} // @name CreateArticleRequest`), and shared types such as
  `httpserver.Problem` do the same, so the API contract never carries a Go package or type name.
- Routes live under `/v1`. Lists take `limit` and `offset` and are ordered newest first. A module's default
  and maximum page sizes are its own settings, 20 and 100 unless configured.

### Code layout

- **One file per unit** in `usecase/` (a use case), `transport/rest/` (an endpoint: its handler, swag
  annotations, request type and own helpers) and `gateway/postgres/` (a repository method). Code that several
  units share stays in `<aggregate>.go` (use cases), `handler.go`, `routes.go`, `response.go`, `errors.go`,
  `<aggregate>_repo.go` and `<aggregate>_row.go`. Types serving one aggregate carry its name: `ArticleUsecase`,
  `ArticleDeps`, `ArticleRepo`. A file is named after its unit (`create.go`, `by_id.go`),
  prefixed with the aggregate when a package serves more than one (`comment_create.go`). Test files mirror
  the source files.
- **Go style** is the ordinary one: gofmt, [Effective Go](https://go.dev/doc/effective_go) and
  [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments); this repository adds no layout or naming
  rules beyond the test conventions below. A name is short when its scope is short and says more the further
  from its declaration it is used; a receiver is one or two letters, the same in every method of its type.
- **Packages are placed by responsibility, not by size:** a technical capability is a platform package named
  after what it does (`clock`, `id`, `requestid`), however small, because it may grow. There is no `util`,
  `common`, `helpers` or `constants` package.
- **Constants live with the code that owns them:**

  | Constant | Where |
  |---|---|
  | A business rule of one module | its `domain/` (`domain.StatusDraft`) |
  | A value other modules need | its `contract/` |
  | A technical value of a platform package | that package (`requestid.Header`) |
  | A problem code | the transport's `errors.go` |
  | A setting's default | the `config.go` that reads the setting |
  | A value one file uses | unexported, in that file |

- **A change to a stored entity** goes through a port method that takes the change as a function and runs
  it inside one transaction holding the entity's lock, as `ArticleRepository.Update(ctx, id, change)` does.
  Never load and save in separate calls: concurrent requests would overwrite each other.
- **Writes that must succeed or fail together** across repositories run in one unit of work: a `UnitOfWork`
  port declared in the use case's `ports.go` hands the use case repositories built on one transaction.
  Repositories take a `database.Querier`, a pool or a transaction. A transaction is never carried in a
  context, and a unit of work never spans two modules ([docs/extending.md](docs/extending.md#units-of-work)).
- **Entities keep their state unexported** and change only through their constructor and methods. A
  gateway rebuilds a stored entity with the domain's `Restore`; nothing else calls it.
- **Use cases return output models**, plain structs such as `usecase.ArticleView`, never entities. A
  transport maps the output model to its own response type.
- **Long-running work** is a `lifecycle.Component`, run by `cmd/server` in a goroutine of its own beside the
  HTTP server: queue consumers, the outbox relay, scheduled jobs and process-local tickers alike. Consumers
  and the relay live in `<module>/transport/event/`, scheduled jobs in `<module>/transport/cron/`; like a
  handler, they call the module's use cases. Work that must not run twice across replicas coordinates
  itself: consumers share a consumer group, the relay claims rows, and an exclusive job runs under a lease. A
  goroutine that outlives a request belongs to a component
  ([docs/extending.md](docs/extending.md#background-work)).

### Logging, tracing and metrics

- Use `log/slog` through the `…Context` methods (`log.InfoContext(ctx, …)`). The logger adds `request_id`,
  `trace_id` and `span_id` from the context and `source` from the call site, so never add them by hand.
- Never log a secret or a submitted value. The logger redacts the value of any key that contains `password`,
  `secret`, `token`, `authorization`, `cookie`, `api_key` or `private_key`. That is a safety net, not a
  permission.
- An error is logged once, as `slog.String("error", err.Error())`, where the response is decided: the
  transport's `errors.go`.
- HTTP requests and SQL queries get spans and metrics without code in the modules. A span around a use case
  or an outgoing call, and a business metric, go where
  [docs/extending.md](docs/extending.md#tracing-with-opentelemetry) shows.
- A metric's attributes have few values: a route pattern, a status, an operation. Never an ID, a path, a
  user or a submitted value; each new value is a new time series.

### Configuration

- Configuration comes from environment variables, and from `./.env` when that file exists; the environment
  wins. A platform variable goes into `internal/platform/config`. A module's variable goes into the module's
  `config.go`, read with a `config.Reader` and named after the module (`ARTICLE_MAX_PAGE_SIZE`). Every
  variable also goes into `.env.example` and the configuration table in `README.md`.
- Every invalid or missing setting is reported at start-up in one error, the platform's and every module's
  together.

### Tests

- **Assertions:** standard library only: `t.Errorf`/`t.Fatalf`, `==`, `reflect.DeepEqual` and `errors.Is`. No
  testify. Failure messages read `<call> = <got>, want <want>`.
- **One test function per unit**, named `Test<Unit>` or `Test<Type><Method>` without underscores
  (`TestFromEnv`, `TestCreate`, `TestArticlePublish`, `TestHandlerList`). Its doc comment states the unit's
  behaviour and starts with the unit's name.
- **Cases** live in `tCases := []struct{ name string; … }` and run through
  `for _, tc := range tCases { t.Run(tc.name, …) }`. Case names are lowercase sentences describing the
  behaviour. A property no single row can show, such as a graceful shutdown, gets its own test function.
- **Mocks** are gomock (`go.uber.org/mock`). The file that declares an interface carries
  `//go:generate go tool mockgen -source=<file>.go -destination=<file>_mock_test.go -package=<pkg>_test`.
  Generated mocks are committed and never edited.
- **Fixtures:** `newFixture(t)` returns a struct with fresh mocks and the unit under test. A case sets its
  expectations through a `mock func(f *fixture)` field; expectations that repeat get named helpers
  (`f.expectFound(article)`). Value fixtures are functions (`articleFixture()`, `articleWith(change)`). Test
  errors are named variables (`errStoreDown`). There is no mutable state at package level.
- **Integration tests** carry the build tag `integration`. `testenv.Postgres(t, files)` gives a test its own
  database, migrated with `files`, inside one container shared by the test binary. Every package with
  integration tests has `func TestMain(m *testing.M) { testenv.Main(m) }` in a file with the same tag.
- **Race detector:** CI runs `-race`; local runs on Windows skip it because it needs cgo.

### Comments

- **Doc comments:** every exported identifier has one, every method of an interface included. It starts with
  the identifier's name and states the contract: what it does, the errors it returns (by name), limits, side
  effects and guarantees such as safety for concurrent use. It does not describe the implementation.
  Unexported identifiers get a comment only when the name is not enough.
- **Inside function bodies:** no comments, except a short reason the code cannot express: a security reason,
  the behaviour of something outside the code, an order that must not change, or a workaround with a link to
  the upstream issue. Tool directives stay: `//go:embed`, `//go:generate`, `//nolint:<linter> // <reason>` and
  swag annotations. A block that seems to need an explanation becomes a function with a clear name.
- **Blank lines inside function bodies:** a blank line separates steps that are not closely related, such as
  reading the input, calling the use case and writing the response. A call and the check of its error stay
  together, as do statements that build one value.
- **Never in a comment:** history ("previously…", "since v0.2.0…"), a restatement of the code,
  commented-out code, a TODO without an issue link, a reference to callers, tasks, plans or tickets ("Used
  by…"), or a credit line.

### Text and commits

- Code, comments, documentation and commit messages are written in English.
- Commit messages read `<type>(<scope>): <title>`, a blank line, then a body of one paragraph that says what
  the commit adds and why, wrapped at 72 columns. Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`. No
  AI or co-author attribution lines.
- While a branch is unpushed, a fix to one of its commits is folded into that commit
  (`git commit --fixup=<sha>`, then `git rebase --autosquash <base>`) instead of being added as a new commit.

## Definition of done

All of these pass:

- `go build ./...` and `go vet ./...`, also with `-tags=integration`;
- `make test`, which includes the architecture test;
- `make test-integration`;
- `make lint`;
- `make generate-check` and `make openapi-check`.

## Never

- Push branches or tags, create or move a tag, or open a pull request without the owner's explicit
  instruction.
- Edit `api/openapi/` or a `*_mock_test.go` file by hand; regenerate them.
- Keep mutable test state at package level: no global mocks and no shared fixtures that tests change.
- Import another module except through its `contract/` package.
