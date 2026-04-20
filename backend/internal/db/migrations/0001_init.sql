CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name     TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('patient','doctor','family')),
    phone         TEXT,
    birth_date    DATE,
    invite_code   TEXT UNIQUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_role ON users(role);

CREATE TABLE patient_links (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    linked_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    relation    TEXT NOT NULL CHECK (relation IN ('doctor','family')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(patient_id, linked_id)
);
CREATE INDEX idx_links_patient ON patient_links(patient_id);
CREATE INDEX idx_links_linked ON patient_links(linked_id);

CREATE TABLE health_metrics (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind          TEXT NOT NULL CHECK (kind IN ('pulse','bp_sys','bp_dia','glucose','temperature','weight','spo2')),
    value         DOUBLE PRECISION NOT NULL,
    value_2       DOUBLE PRECISION,
    note          TEXT,
    measured_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_metrics_patient_kind_time ON health_metrics(patient_id, kind, measured_at DESC);

CREATE TABLE alerts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    metric_id     UUID REFERENCES health_metrics(id) ON DELETE SET NULL,
    severity      TEXT NOT NULL CHECK (severity IN ('info','warning','critical')),
    reason        TEXT NOT NULL,
    kind          TEXT NOT NULL,
    value         DOUBLE PRECISION,
    baseline_mean DOUBLE PRECISION,
    baseline_std  DOUBLE PRECISION,
    acknowledged  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_alerts_patient_time ON alerts(patient_id, created_at DESC);

CREATE TABLE medications (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    dosage        TEXT,
    times_of_day  TEXT[] NOT NULL DEFAULT '{}',
    start_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date      DATE,
    active        BOOLEAN NOT NULL DEFAULT TRUE,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_meds_patient ON medications(patient_id, active);

CREATE TABLE medication_logs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    medication_id  UUID NOT NULL REFERENCES medications(id) ON DELETE CASCADE,
    patient_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scheduled_at   TIMESTAMPTZ NOT NULL,
    status         TEXT NOT NULL CHECK (status IN ('taken','missed','skipped')),
    logged_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(medication_id, scheduled_at)
);
CREATE INDEX idx_med_logs_patient_time ON medication_logs(patient_id, scheduled_at DESC);

CREATE TABLE messages (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body         TEXT NOT NULL,
    read_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_messages_thread ON messages(
    LEAST(sender_id, recipient_id),
    GREATEST(sender_id, recipient_id),
    created_at DESC
);
