DROP INDEX IF EXISTS idx_services_domain;
ALTER TABLE services DROP COLUMN IF EXISTS domain;
