ALTER TABLE permissions ADD COLUMN IF NOT EXISTS resource VARCHAR(100);

ALTER TABLE permissions DROP CONSTRAINT IF EXISTS uq_permissions_service_bit;

CREATE UNIQUE INDEX IF NOT EXISTS uq_permissions_service_resource_bit 
ON permissions(service_id, COALESCE(resource, ''), bit);

-- Add index for resource
CREATE INDEX IF NOT EXISTS idx_permissions_resource ON permissions(resource);

