INSERT INTO permissions (service_id, bit, name, active, resource)
VALUES 
    ('00000000-0000-0000-0000-000000000001', 4, 'manage_services', true, NULL),
    ('00000000-0000-0000-0000-000000000001', 5, 'manage_permissions', true, NULL),
    ('00000000-0000-0000-0000-000000000001', 6, 'manage_users', true, NULL),
    ('00000000-0000-0000-0000-000000000001', 7, 'manage_roles', true, NULL),
    ('00000000-0000-0000-0000-000000000001', 8, 'system_admin', true, NULL)
ON CONFLICT (service_id, COALESCE(resource, ''), bit) DO NOTHING;

