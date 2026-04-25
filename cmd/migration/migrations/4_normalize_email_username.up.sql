-- Email is normalized to lowercase at the usecase boundary; bring any
-- pre-existing rows into the same shape so the partial unique index
-- on `email` keeps doing case-insensitive duplicate detection.
UPDATE users
SET email = LOWER(TRIM(email))
WHERE email IS DISTINCT FROM LOWER(TRIM(email));

-- Username uniqueness should be case-insensitive too — "Patrick" and
-- "patrick" can't both exist. We do NOT lowercase existing rows
-- (display case is meaningful); instead the unique index sits on
-- LOWER(username), filtered to active rows.
DROP INDEX IF EXISTS idx_users_username_active;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_active_ci
    ON users (LOWER(username))
    WHERE deleted_at IS NULL;
