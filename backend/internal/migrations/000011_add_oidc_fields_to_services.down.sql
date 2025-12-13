UPDATE services 
SET code = SUBSTRING(client_id FROM '^authly_([^_]+)_')
WHERE code IS NULL;
ALTER TABLE services ALTER COLUMN code SET NOT NULL;
CREATE UNIQUE INDEX IF EXISTS idx_services_code ON services(code);

DROP INDEX IF EXISTS idx_services_client_id;
ALTER TABLE services DROP COLUMN IF EXISTS allowed_scopes;
ALTER TABLE services DROP COLUMN IF EXISTS redirect_uris;
ALTER TABLE services DROP COLUMN IF EXISTS client_secret;
ALTER TABLE services DROP COLUMN IF EXISTS client_id;
