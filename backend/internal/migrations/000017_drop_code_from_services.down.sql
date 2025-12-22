ALTER TABLE services ADD COLUMN IF NOT EXISTS code VARCHAR(50);

UPDATE services 
SET code = SUBSTRING(client_id FROM '^authly_([^_]+)_')
WHERE code IS NULL AND client_id LIKE 'authly_%';

UPDATE services 
SET code = 'unknown_' || SUBSTRING(REPLACE(id::text, '-', ''), 1, 8)
WHERE code IS NULL;

ALTER TABLE services ALTER COLUMN code SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_services_code ON services(code);
