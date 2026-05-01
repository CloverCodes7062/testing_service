ALTER TABLE incidents
    ADD CONSTRAINT IF NOT EXISTS incidents_severity_check CHECK (severity BETWEEN 1 AND 5),
    ADD CONSTRAINT IF NOT EXISTS incidents_status_check CHECK (status IN ('open', 'investigating', 'resolved', 'closed'));

-- Data backfill: insert default tags
INSERT INTO tags (name) VALUES ('bug'), ('outage'), ('performance'), ('security'), ('maintenance')
ON CONFLICT (name) DO NOTHING;
