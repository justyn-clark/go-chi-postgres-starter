-- Drop indexes
DROP INDEX IF EXISTS idx_users_password_reset_token;
DROP INDEX IF EXISTS idx_users_role;

-- Drop columns
ALTER TABLE users DROP COLUMN IF EXISTS password_reset_expires_at;
ALTER TABLE users DROP COLUMN IF EXISTS password_reset_token;
ALTER TABLE users DROP COLUMN IF EXISTS role;

