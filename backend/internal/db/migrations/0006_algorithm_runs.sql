CREATE TABLE IF NOT EXISTS algorithm_runs (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    metric_id          UUID NOT NULL REFERENCES health_metrics(id) ON DELETE CASCADE,
    kind               TEXT NOT NULL,
    value              DOUBLE PRECISION NOT NULL,
    estimator          TEXT NOT NULL,
    mean_used          DOUBLE PRECISION,
    std_used           DOUBLE PRECISION,
    z_score            DOUBLE PRECISION,
    severity           TEXT NOT NULL,
    reason_code        TEXT NOT NULL,
    used_history       BOOLEAN NOT NULL,
    history_size       INTEGER NOT NULL,
    algorithm_version  TEXT NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_algorithm_runs_patient_kind ON algorithm_runs(patient_id, kind, created_at DESC);
