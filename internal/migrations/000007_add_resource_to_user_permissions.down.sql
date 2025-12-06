
DROP INDEX IF EXISTS idx_user_permissions_resource;
DROP INDEX IF EXISTS uq_user_permissions_user_service_resource;

ALTER TABLE user_permissions ADD CONSTRAINT uq_user_permissions_user_service 
UNIQUE (user_id, service_id);

ALTER TABLE user_permissions DROP COLUMN IF EXISTS resource;

