-- FK indexes
CREATE INDEX IF NOT EXISTS idx_incidents_team_id ON incidents(team_id);
CREATE INDEX IF NOT EXISTS idx_team_members_team_id ON team_members(team_id);
CREATE INDEX IF NOT EXISTS idx_team_members_user_id ON team_members(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_incident_id ON comments(incident_id);
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
CREATE INDEX IF NOT EXISTS idx_incident_tags_incident_id ON incident_tags(incident_id);
CREATE INDEX IF NOT EXISTS idx_incident_tags_tag_id ON incident_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);

-- Partial index on open/investigating incidents
CREATE INDEX IF NOT EXISTS idx_incidents_active ON incidents(created_at)
    WHERE status IN ('open', 'investigating');

-- GIN index for full-text search
CREATE INDEX IF NOT EXISTS idx_incidents_fts ON incidents
    USING GIN (to_tsvector('english', title || ' ' || description));

-- Composite index
CREATE INDEX IF NOT EXISTS idx_incidents_status_severity ON incidents(status, severity);

-- Covering index
CREATE INDEX IF NOT EXISTS idx_incidents_status_covering ON incidents(status)
    INCLUDE (id, title, created_at);
