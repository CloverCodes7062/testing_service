INSERT INTO users (name, email, role, password_hash) VALUES
    ('Admin User', 'admin@example.com', 'admin', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQyCqHRHEhSoRnBQbEtblQLFS'),
    ('Manager User', 'manager@example.com', 'manager', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQyCqHRHEhSoRnBQbEtblQLFS'),
    ('Member User', 'member@example.com', 'member', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQyCqHRHEhSoRnBQbEtblQLFS')
ON CONFLICT (email) DO NOTHING;

INSERT INTO teams (name) VALUES ('Platform'), ('Backend'), ('Frontend')
ON CONFLICT DO NOTHING;

INSERT INTO incidents (title, description, status, severity)
SELECT 'Sample Incident ' || n, 'Description for incident ' || n,
    (ARRAY['open','investigating','resolved','closed'])[1 + (n % 4)],
    1 + (n % 5)
FROM generate_series(1, 5) AS n
ON CONFLICT DO NOTHING;
