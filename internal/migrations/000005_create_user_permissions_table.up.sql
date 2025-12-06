CREATE TABLE IF NOT EXISTS user_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    user_id UUID NOT NULL,
    service_id UUID NOT NULL,
    bitmask NUMERIC(20) NOT NULL DEFAULT 0,
    permission_v INTEGER NOT NULL DEFAULT 1,
    CONSTRAINT fk_user_permissions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_permissions_service FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE,
    CONSTRAINT uq_user_permissions_user_service UNIQUE (user_id, service_id)
);

CREATE INDEX IF NOT EXISTS idx_user_permissions_deleted_at ON user_permissions(deleted_at);
CREATE INDEX IF NOT EXISTS idx_user_permissions_user_id ON user_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_permissions_service_id ON user_permissions(service_id);
CREATE INDEX IF NOT EXISTS idx_user_permissions_permission_v ON user_permissions(permission_v);

