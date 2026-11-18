# Modules

Each folder here is one feature module: everything a feature needs, from its rules to its HTTP endpoints
and its tables. Modules are independent of each other; `internal/app` puts them together.

## Anatomy

```
<name>/
├── README.md               # the module card
├── module.go               # Deps, New, RegisterRoutes: the only file that names concrete gateways
├── config.go               # Config, LoadConfig: the module's own <NAME>_* settings (when it has any)
├── api.go                  # API, what other modules may call (when they call it)
├── contract/               # the API's types and the events the module publishes (standard library only)
├── domain/                 # entities with unexported state, value objects, rules, sentinel errors (standard library only)
├── usecase/
│   ├── ports.go            # every outbound need, as interfaces (+ generated ports_mock_test.go)
│   ├── <aggregate>.go      # ArticleUsecase, ArticleDeps, NewArticleUsecase, the output models they return
│   └── <case>.go           # one file per use case, with its input and output types (+ <case>_test.go)
├── transport/rest/
│   ├── usecases.go         # UseCases: the interface the handlers call (+ generated usecases_mock_test.go)
│   ├── handler.go          # Handler, NewHandler, helpers several endpoints share
│   ├── routes.go           # the module's endpoints, relative to the groups they are mounted on
│   ├── <endpoint>.go       # one file per endpoint: handler, swag annotations, request type (+ <endpoint>_test.go)
│   ├── response.go         # response bodies several endpoints share
│   └── errors.go           # domain errors mapped to problem codes
└── gateway/postgres/       # an adapter across the module's boundary: the database (see Gateways below)
    ├── <aggregate>_repo.go # the repository type, its constructor, driver errors turned into domain errors
    ├── <aggregate>_row.go  # row structs with db tags and their mapping to entities
    └── <method>.go         # one file per repository method (+ <method>_integration_test.go)
```

A file is named after its unit. When a package serves more than one aggregate, the name starts with the
aggregate (`comment_create.go`).

`transport/rest/` is one package on purpose. `routes.go` is its only route map, and each endpoint's file
holds its handler and its request type. Request and response types are unexported, since nothing outside
the package uses them, and each names its OpenAPI schema with swag's `// @name`. Splitting it by kind (`handlers/`, `dto/`, `routes/`) would export
every handler, rename the OpenAPI schemas and spread one endpoint over several packages. When a module
serves several resources, split it by resource instead (`transport/rest/comment/`), each part with its own
`routes.go`.

| Part | Holds | Never holds |
|---|---|---|
| `domain/` | entities, invariants, sentinel errors | exported mutable state, `json` or `db` tags, imports outside the standard library |
| `usecase/` | the use cases, their input and output models, and the ports they need | SQL, HTTP, platform packages, entities in its results |
| `transport/rest/` | HTTP decoding, validation, error mapping, DTOs | SQL, business rules |
| `gateway/postgres/` | SQL, row mapping, driver errors turned into domain errors | HTTP, business rules |
| `module.go`, `config.go` | wiring, the module's settings | logic |

**Gateways.** `gateway/` holds only adapters that cross the module's boundary: external systems (the database,
a cache, a broker, mail, an HTTP API), another module (its anti-corruption layer over that module's `api.go`),
and the events the module publishes. A technical capability (time, identifiers, password hashing, randomness,
token signing) comes from a platform package whose type satisfies the use case's port directly, as
`clock.Real` and `id.Generator` do in `article/module.go`; it needs no gateway.

**Optional parts.** A module without business rules may skip `domain/`. A module that no other module calls
has no `contract/` and no `api.go`. `article` has both, so the examples can build on it.

## The layers as Clean Architecture

The folders are the rings of Clean Architecture and the ports and adapters of Hexagonal Architecture.
Dependencies point inwards, and `internal/archtest` checks them.

| Clean Architecture | Hexagonal | Here |
|---|---|---|
| Entities (enterprise business rules) | domain model | `domain/` |
| Use case interactors (application business rules) | application | `usecase/`: `<aggregate>.go` and one file per use case |
| Input port, input and output data | driving port | the exported methods of `usecase.ArticleUsecase`, with the input and output models beside them |
| Data access interfaces, gateways' interfaces | driven ports | `usecase/ports.go` |
| Controllers and presenters | driving adapters | `transport/rest/` (later `transport/event/`, `transport/cron/`, `transport/grpc/`) |
| Gateways | driven adapters | `gateway/postgres/`, `gateway/<provider>/` |
| Frameworks and drivers | technology | `internal/platform/`, the libraries, `cmd/` |
| Main component | configuration | `module.go`, `internal/app` |

**Rules that keep the rings honest**
- **Entities guard their own state.** Their fields are unexported, and they change only through their
  constructor and methods, which check the rules.
  - A gateway loading a stored entity rebuilds it with the domain's `Restore`. `Restore` exists only for
    gateways and does not check the rules again, so data stored under an older rule still loads.
- **Only plain data crosses the use-case boundary.**
  - Use cases take input models (`CreateInput`) and return output models (`ArticleView`), never entities.
  - A transport therefore cannot call domain behaviour outside a use case.
  - The entity can change shape without touching any adapter.
- **Ports belong to the use cases** that need them. Gateways implement them, and they never return driver
  types.

**Deliberate differences from the textbook diagrams**
- **No input-port interface in the use-case ring.**
  - The input port is the concrete API of `usecase.ArticleUsecase`, as Go advises for providers.
  - Each transport declares the small interface it calls, for its mocks. Interfaces belong to the consumer in Go.
- **No presenter or output-port interface.** Use cases return output models, and the transport presents them.
  Every Go reference does the same.
- **One use case type per aggregate, one file per use case.** One handler type per use case pays off when generic
  decorators are wanted ([docs/extending.md](../../docs/extending.md)).

## Ports (`usecase/ports.go`)

- The use cases decide the shape of each interface: one method per need, not one per table operation.
- Signatures use only domain types, types declared in `ports.go`, and the standard library.
- Writes that must succeed or fail together across repositories go through a `UnitOfWork` port that hands out
  repositories built on one transaction ([docs/extending.md](../../docs/extending.md#units-of-work)).
- Every method's doc comment states its contract, including the domain errors it returns. An implementer of
  the port has nothing else to go on.
- A change to a stored entity is passed in as a function, which the store runs inside one transaction holding
  the entity's lock, as in `ArticleRepository.Update(ctx, id, change)`. Loading, changing and saving in
  separate calls lets concurrent requests overwrite each other.
- A method with no caller is deleted.
- The file carries the `//go:generate go tool mockgen …` line for its mocks.

## Creating a module

1. Write the domain: the entity with unexported fields, its constructor, accessors and methods that keep its
   rules, `Restore` for loading it, and its sentinel errors. Test it with table tests.
2. Write `usecase/ports.go`, run `make generate`, then write the use cases test first against the mocks. A
   use case returns an output model built from the entity, never the entity.
3. Write the gateway on a `database.Querier` and its integration tests with
   `testenv.Postgres(t, migrations.FS)`. Translate driver errors (unique violations, malformed IDs, no rows)
   into domain errors.
4. Create the table: `make migrate-create name=<name>_<action>`, then fill in `-- +goose Up` and
   `-- +goose Down`.
5. Write the transport: the `UseCases` interface, one file per endpoint with its swag annotations and
   request type, `routes.go`, and the error mapping. Test it with mocks of `UseCases`.
6. Write `module.go`, and `config.go` when the module has settings. Build the module in
   `internal/app/modules.go`, add its settings to `Config` and `LoadConfig` in `internal/app/config.go`, and
   mount it in `internal/app/routes.go`, on the `/v1` group or on a group with its own middleware.
7. Write the module card, run `make generate openapi`, and go through the definition of done in
   [AGENTS.md](../../AGENTS.md).

The `article` module is the reference: [article/README.md](article/README.md).

## Between modules

- Another module is reached only through its `contract/` package (plain data, events and published errors,
  standard library only) and its `api.go` (facade methods that return contract types).
- Only `transport/` (event subscribers) and `gateway/` import another module's `contract/`. The consumer
  declares a small interface over the provider's `api.go` in its own `gateway/<provider>/`, an
  anti-corruption layer that keeps the provider's model out of the consumer's domain.
- Module dependencies never form a cycle; the reverse direction uses events.
- Modules never share tables, and no foreign key points from one module's table to another's.

`internal/archtest` enforces the import rules: who may import another module, and through which package.
Cycles between modules, shared tables and cross-module foreign keys are checked in review.
