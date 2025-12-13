DROP INDEX IF EXISTS idx_authorization_codes_used;
DROP INDEX IF EXISTS idx_authorization_codes_expires_at;
DROP INDEX IF EXISTS idx_authorization_codes_user_id;
DROP INDEX IF EXISTS idx_authorization_codes_client_id;
DROP INDEX IF EXISTS idx_authorization_codes_code;
DROP INDEX IF EXISTS idx_authorization_codes_deleted_at;

DROP TABLE IF EXISTS authorization_codes;

