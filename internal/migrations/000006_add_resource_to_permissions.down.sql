DROP INDEX IF EXISTS uq_permissions_service_resource_bit;

ALTER TABLE permissions ADD CONSTRAINT uq_permissions_service_bit 
UNIQUE (service_id, bit);

ALTER TABLE permissions DROP COLUMN IF EXISTS resource;

