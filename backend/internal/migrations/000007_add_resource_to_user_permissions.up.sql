ALTER TABLE user_permissions ADD COLUMN IF NOT EXISTS resource VARCHAR(100);

ALTER TABLE user_permissions DROP CONSTRAINT IF EXISTS uq_user_permissions_user_service;

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_permissions_user_service_resource 
ON user_permissions(user_id, service_id, COALESCE(resource, ''));

CREATE INDEX IF NOT EXISTS idx_user_permissions_resource ON user_permissions(resource);

