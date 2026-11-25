-- +goose Up
CREATE TABLE comments (
    id          uuid        PRIMARY KEY,
    article_id  uuid        NOT NULL,
    author_id   uuid        NOT NULL,
    author_name text        NOT NULL,
    body        text        NOT NULL,
    created_at  timestamptz NOT NULL
);
CREATE INDEX comments_article_id_created_at_idx ON comments (article_id, created_at DESC, id DESC);

-- +goose Down
DROP TABLE comments;
