CREATE TABLE IF NOT EXISTS players(
    id uuid PRIMARY KEY,
    installation_id uuid NOT NULL UNIQUE,
    email VARCHAR(50) UNIQUE,
    password VARCHAR(255),
    role_id smallint NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz,
    deleted_at timestamptz,
    password_changed_at timestamptz,
);