CREATE TABLE IF NOT EXISTS audit_log (
    id            BIGSERIAL PRIMARY KEY,
    actor_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_role    TEXT,
    patient_id    UUID REFERENCES users(id) ON DELETE CASCADE,
    method        TEXT NOT NULL,
    path          TEXT NOT NULL,
    status        INTEGER NOT NULL,
    ip            TEXT,
    user_agent    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_patient ON audit_log(patient_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_actor ON audit_log(actor_id, created_at DESC);
