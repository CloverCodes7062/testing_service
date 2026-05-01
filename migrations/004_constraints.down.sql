ALTER TABLE incidents
    DROP CONSTRAINT IF EXISTS incidents_severity_check,
    DROP CONSTRAINT IF EXISTS incidents_status_check;
