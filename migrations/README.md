# Migrations

The SQL migrations of every module. `migrations.go` embeds them into the `migrate` binary (and the
integration tests), so the container image needs no migration files.

## Naming

`YYYYMMDDHHMMSS_<module>_<action>.sql`, for example `20261118035840_article_create_articles.sql`. The
timestamp is the version. Create a file with:

```sh
make migrate-create name=<module>_<action>
```

Each file holds a `-- +goose Up` section and a `-- +goose Down` section that undoes it.

## Rules

- One file changes one module's tables.
- No foreign key points from one module's table to another module's table; modules refer to each other by
  ID only.
- Never edit a migration that has been applied anywhere. Write a new one that changes the schema further.
- Versions apply in order, and a version older than one already applied still applies. A migration merged
  from another branch with an earlier timestamp is therefore never skipped. Keep migrations independent of
  each other's order where you can.

## Running

```sh
make migrate-up       # apply every pending migration
make migrate-down     # roll back the latest one
make migrate-status   # list them and when each was applied
```

These read `DATABASE_URL` from the environment or `./.env`. In `deployment/compose.yml` the `migrate` service
runs `migrate up` once before the API starts.
