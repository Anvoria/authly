CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    service_id UUID NOT NULL,
    bit SMALLINT NOT NULL CHECK (bit >= 0 AND bit < 64),
    name VARCHAR(100) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT fk_permissions_service FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE,
    CONSTRAINT uq_permissions_service_bit UNIQUE (service_id, bit)
);

CREATE INDEX IF NOT EXISTS idx_permissions_deleted_at ON permissions(deleted_at);
CREATE INDEX IF NOT EXISTS idx_permissions_service_id ON permissions(service_id);
CREATE INDEX IF NOT EXISTS idx_permissions_active ON permissions(active);

INSERT INTO permissions (service_id, bit, name, active)
VALUES 
    ('00000000-0000-0000-0000-000000000001', 0, 'read', true),
    ('00000000-0000-0000-0000-000000000001', 1, 'write', true),
    ('00000000-0000-0000-0000-000000000001', 2, 'delete', true),
    ('00000000-0000-0000-0000-000000000001', 3, 'admin', true)
ON CONFLICT (service_id, bit) DO NOTHING;

