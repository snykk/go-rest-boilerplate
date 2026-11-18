# Module: article

The reference module. An article is a piece of writing addressed by a unique slug. It is created as a
draft, can be edited, and is published once.

## Endpoints

All under `/v1`; errors are RFC 9457 problems. `RegisterRoutes(read, write)` puts the public reads on `read`,
and the changes and the reads of drafts on `write`, so a service can put authentication on `write` alone; the
core mounts both on `/v1`.

| Group | Method | Path | Success | Errors |
|---|---|---|---|---|
| write | POST | `/articles` | 201, `Location: /v1/articles/drafts/{id}` | 400 `malformed_json`, 409 `slug_taken`, 413 `body_too_large`, 422 `validation_failed`, `invalid_slug`, `invalid_title`, `invalid_body`, `body_too_long` |
| read | GET | `/articles?limit=&offset=` | 200, a page of published articles, newest first | 400 `invalid_query` |
| read | GET | `/articles/{id}` | 200, a published article | 404 `not_found`, also for a draft |
| write | GET | `/articles/drafts?limit=&offset=` | 200, a page of drafts, newest first | 400 `invalid_query` |
| write | GET | `/articles/drafts/{id}` | 200, a draft | 404 `not_found`, also for a published article |
| write | PATCH | `/articles/{id}` | 200 | 400 `malformed_json`, 404 `not_found`, 409 `slug_taken`, `slug_locked`, 413 `body_too_large`, 422 `invalid_slug`, `invalid_title`, `invalid_body`, `body_too_long` |
| write | POST | `/articles/{id}/publish` | 200 | 404 `not_found`, 409 `already_published`, 422 `empty_body` |
| write | DELETE | `/articles/{id}` | 204 | 404 `not_found` |

An `{id}` that is not a UUID answers 404 without reaching the database.

## Settings

Read by `config.go`; the application reports an invalid value at start-up.

| Variable | Default | Notes |
|---|---|---|
| `ARTICLE_DEFAULT_PAGE_SIZE` | `20` | Articles in a list that asks for no `limit` |
| `ARTICLE_MAX_PAGE_SIZE` | `100` | The most articles a list returns; a larger `limit` is lowered to it |

## Rules

- `slug`: 1–100 characters matching `^[a-z0-9]+(-[a-z0-9]+)*$`, unique, and fixed once the article is
  published.
- `title`: trimmed, 1–200 characters, none of them NUL.
- `body`: at most 50 000 characters, none of them NUL; it must not be blank to publish.
- `status`: `draft` on creation; publishing makes it `published`, which is final.
- `published_at` is set when the article is published. All times are UTC.

## Table: `articles`

| Column | Type | Notes |
|---|---|---|
| `id` | `uuid` | primary key, UUIDv7 |
| `slug` | `text` | unique index `articles_slug_key` |
| `title` | `text` | |
| `body` | `text` | default `''` |
| `status` | `text` | `draft` or `published` |
| `published_at` | `timestamptz` | null until published |
| `created_at`, `updated_at` | `timestamptz` | |

Index `articles_status_created_at_idx` serves the list filtered by status.

## Ports

`usecase/ports.go` declares what the use cases need: `ArticleRepository` (implemented by
`gateway/postgres.ArticleRepo`), `Clock` (`platform/clock.Real`) and `IDGenerator` (`platform/id.Generator`).

Edits and publishing go through `ArticleRepository.Update`, which applies the change inside one transaction
that holds the article's row lock. Concurrent requests to one article therefore apply one after the other,
and none of them overwrites another.

`domain.Article` keeps its fields unexported; `gateway/postgres` rebuilds a stored article with
`domain.Restore`. The use cases return `usecase.ArticleView`, plain data built from the entity, and the
transport maps it to its own unexported `articleResponse`. No adapter holds the entity.

## Public API and events

- `contract/`: `Article` (ID, slug, title, whether it is published) and `ErrNotFound`. Standard library only.
- `api.go`: `(*Module).API()` returns `API`, whose `Article(ctx, id)` looks an article up whatever its status.
  A consumer wraps it in its own `gateway/article/`.
- Events: none.

## Dependencies

No other module.

## Reading order

1. `domain/`: `article.go` (the entity, its accessors and `Restore`), then `errors.go`.
2. `usecase/ports.go`, `article.go` (`ArticleUsecase` and `ArticleView`), then one use case, such as `update.go`.
3. `gateway/postgres/`: `article_repo.go`, `article_row.go`, then one method, such as `update.go`.
4. `transport/rest/`: `routes.go`, `handler.go`, then one endpoint, such as `update.go`, and `errors.go`.
5. `contract/contract.go` and `api.go`: what other modules see.
6. `config.go`, then `module.go`.
