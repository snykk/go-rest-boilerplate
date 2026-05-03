DROP INDEX IF EXISTS idx_users_username_active_ci;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_active
    ON users (username)
    WHERE deleted_at IS NULL;
-- Existing email rows stay lowercased; original case isn't recoverable.
