# Module: comment

Comments on published articles. A signed-in user comments on an article once it is published; anyone reads the
comments; the comment's author or an editor deletes one. The module asks the `article` module whether an
article is published and the `identity` module for the author's display name, each through that module's API.

## Endpoints

All under `/v1`; errors are RFC 9457 problems. `RegisterRoutes(public, signedIn)` puts the list on `public`
and the two endpoints that act for the caller on `signedIn`, which `internal/app` builds with
`auth.Authenticate` over the identity module's access tokens.

| Group | Method | Path | Success | Errors |
|---|---|---|---|---|
| signedIn | POST | `/comments` (`{article_id, body}`) | 201, the comment | 400 `malformed_json`, 401 `unauthenticated`, 404 `not_found`, 413 `body_too_large`, 422 `validation_failed` |
| public | GET | `/comments?article_id=&author_id=&limit=&offset=` | 200, a page of the article's comments, newest first | 400 `invalid_query`, 404 `not_found` |
| signedIn | DELETE | `/comments/{id}` | 204 | 401 `unauthenticated`, 403 `forbidden`, 404 `not_found` |

- `article_id` is required. Missing or not a UUID, it is 422 `validation_failed` in a body and 400
  `invalid_query` in a query; an `author_id` that is not a UUID is 400 `invalid_query` too. No problem repeats
  the value.
- An article that does not exist, or is a draft, is 404 `not_found` both to comment on and to list: its
  comments are never an empty page. A caller whose account is gone gets 404 `not_found` when commenting.
- A `body` the rules refuse is a `validation_failed` problem that names the field `body` and never repeats it.
- An `{id}` that is not a UUID answers 404 without reaching the database.

Comments live under `/comments`, not `/articles/{id}/comments`: `ServeMux` refuses `GET /articles/{id}/comments`
beside the article module's `GET /articles/drafts/{id}`, since a path such as `/articles/drafts/comments`
matches both and neither is more specific.

## Settings

Read by `config.go`; the application reports an invalid value at start-up.

| Variable | Default | Notes |
|---|---|---|
| `COMMENT_DEFAULT_PAGE_SIZE` | `20` | Comments in a list that asks for no `limit` |
| `COMMENT_MAX_PAGE_SIZE` | `100` | The most comments a list returns; a larger `limit` is lowered to it |

## Rules

- `body`: trimmed, 1–2000 characters, none of them NUL.
- A comment is written by the signed-in caller, on a published article, and keeps the display name its author
  had then. It never changes afterwards.
- The caller who wrote a comment may delete it, and so may anyone with the `editor` role; anyone else gets
  `forbidden`. The rule sits in the `Delete` use case; the transport passes on who calls and whether they are an
  editor.
- `DeleteForArticle` removes every comment on an article; calling it again changes nothing. All times are UTC.

## Table: `comments`

| Column | Type | Notes |
|---|---|---|
| `id` | `uuid` | primary key, UUIDv7 |
| `article_id` | `uuid` | the article's ID, a plain column: no foreign key to the `article` module's table |
| `author_id` | `uuid` | the author's user ID, a plain column: no foreign key to the `identity` module's table |
| `author_name` | `text` | the author's display name when the comment was written |
| `body` | `text` | |
| `created_at` | `timestamptz` | |

Index `comments_article_id_created_at_idx` serves the list of one article's comments, newest first.

## Ports

`usecase/ports.go` declares what the use cases need: `CommentRepository` (implemented by
`gateway/postgres.CommentRepo`), `Articles` (`gateway/article.Articles`), `Authors`
(`gateway/identity.Authors`), `Clock` (`platform/clock.Real`) and `IDGenerator` (`platform/id.Generator`).

`domain.Comment` keeps its fields unexported; `gateway/postgres` rebuilds a stored comment with
`domain.Restore`. The use cases return `usecase.CommentView`, and the transport maps it to its own unexported
`commentResponse`.

## Dependencies

- `article`, through `gateway/article/`: an anti-corruption layer over `article.API`, behind its own small `API`
  interface. `Published(ctx, id)` is true for a published article, and false, with no error, for a draft and
  for an ID the article module reports as `contract.ErrNotFound`.
- `identity`, through `gateway/identity/`: the same over `identity.API`. `DisplayName(ctx, userID)` turns
  `contract.ErrNotFound` into `domain.ErrAuthorNotFound`.
- The transport reads the `editor` role name from `identity/contract`.

No module depends on `comment`.

## Reading order

1. `domain/`: `comment.go` (the entity, its accessors and `Restore`), then `errors.go`.
2. `usecase/ports.go`, `comment.go` (`CommentUsecase` and `CommentView`), then `add.go` and `delete.go`.
3. `gateway/article/` and `gateway/identity/`: the two anti-corruption layers.
4. `gateway/postgres/`: `comment_repo.go`, `comment_row.go`, then one method, such as `list.go`.
5. `transport/rest/`: `routes.go`, `handler.go`, then one endpoint, such as `add.go`, and `errors.go`.
6. `config.go`, then `module.go`.
