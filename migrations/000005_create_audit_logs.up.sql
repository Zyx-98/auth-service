CREATE TABLE audit_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type   VARCHAR(50)  NOT NULL,
    actor_id     UUID         REFERENCES users(id) ON DELETE SET NULL,
    resource_id  UUID,
    resource_type VARCHAR(50),
    action       VARCHAR(50)  NOT NULL,
    status       VARCHAR(20)  NOT NULL,
    status_reason VARCHAR(255),
    old_value    JSONB,
    new_value    JSONB,
    metadata     JSONB,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_actor_id    ON audit_logs(actor_id, created_at DESC);
CREATE INDEX idx_audit_logs_resource_id ON audit_logs(resource_id, created_at DESC);
CREATE INDEX idx_audit_logs_event_type  ON audit_logs(event_type, created_at DESC);
