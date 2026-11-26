-- +goose Up
CREATE TABLE articles (
    id           uuid        PRIMARY KEY,
    slug         text        NOT NULL,
    title        text        NOT NULL,
    body         text        NOT NULL DEFAULT '',
    status       text        NOT NULL CHECK (status IN ('draft', 'published')),
    published_at timestamptz,
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL
);
CREATE UNIQUE INDEX articles_slug_key ON articles (slug);
CREATE INDEX articles_status_created_at_idx ON articles (status, created_at DESC);

-- +goose Down
DROP TABLE articles;
