-- Switch username/email uniqueness to partial indexes that ignore
-- soft-deleted rows. Without this, re-registering a previously
-- deleted email fails on the full-column UNIQUE constraint.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_active  ON users(email)    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_active ON users(username) WHERE deleted_at IS NULL;
