ALTER TABLE alerts
    ADD COLUMN IF NOT EXISTS reason_code        TEXT NOT NULL DEFAULT 'legacy',
    ADD COLUMN IF NOT EXISTS algorithm_version  TEXT NOT NULL DEFAULT 'v1';

CREATE INDEX IF NOT EXISTS idx_alerts_reason_code ON alerts(reason_code);
