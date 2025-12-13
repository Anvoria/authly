ALTER TABLE services ADD COLUMN IF NOT EXISTS client_id VARCHAR(255);
ALTER TABLE services ADD COLUMN IF NOT EXISTS client_secret VARCHAR(255);
ALTER TABLE services ADD COLUMN IF NOT EXISTS redirect_uris TEXT[] DEFAULT '{}';
ALTER TABLE services ADD COLUMN IF NOT EXISTS allowed_scopes TEXT[] DEFAULT '{}';

UPDATE services 
SET 
    client_id = 'authly_' || code || '_' || SUBSTRING(REPLACE(id::text, '-', ''), 1, 8),
    client_secret = SUBSTRING(MD5(RANDOM()::text || id::text || NOW()::text), 1, 32)
WHERE client_id IS NULL OR client_secret IS NULL;

ALTER TABLE services ALTER COLUMN client_id SET NOT NULL;
ALTER TABLE services ALTER COLUMN client_secret SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_services_client_id ON services(client_id);
