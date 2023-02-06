CREATE TABLE IF NOT EXISTS users(
    id uuid PRIMARY KEY,
    username VARCHAR(25) NOT NULL UNIQUE,
    email VARCHAR(50) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    active BOOLEAN NOT NULL,
    created_at timestamptz  NOT NULL,
    updated_at timestamptz,
    deleted_at timestamptz 
);