DROP INDEX IF EXISTS idx_users_email_active;
DROP INDEX IF EXISTS idx_users_username_active;

ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
ALTER TABLE users ADD CONSTRAINT users_username_key UNIQUE (username);
