-- +goose Up
CREATE TABLE users (
    id            uuid        PRIMARY KEY,
    email         text        NOT NULL,
    display_name  text        NOT NULL,
    password_hash text        NOT NULL,
    roles         text[]      NOT NULL DEFAULT '{}',
    verified_at   timestamptz,
    failed_logins integer     NOT NULL DEFAULT 0,
    locked_until  timestamptz,
    created_at    timestamptz NOT NULL,
    updated_at    timestamptz NOT NULL
);
CREATE UNIQUE INDEX users_email_key ON users (lower(email));

CREATE TABLE sessions (
    id         uuid        PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    family     uuid        NOT NULL,
    token_hash text        NOT NULL,
    expires_at timestamptz NOT NULL,
    spent_at   timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL
);
CREATE UNIQUE INDEX sessions_token_hash_key ON sessions (token_hash);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_family_idx ON sessions (family);

CREATE TABLE email_verifications (
    user_id    uuid        PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    hash       text        NOT NULL,
    expires_at timestamptz NOT NULL,
    attempts   integer     NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL
);

CREATE TABLE password_resets (
    user_id    uuid        PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    hash       text        NOT NULL,
    expires_at timestamptz NOT NULL,
    attempts   integer     NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL
);
CREATE UNIQUE INDEX password_resets_hash_key ON password_resets (hash);

-- +goose Down
DROP TABLE password_resets;
DROP TABLE email_verifications;
DROP TABLE sessions;
DROP TABLE users;
