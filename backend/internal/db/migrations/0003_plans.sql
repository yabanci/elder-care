CREATE TABLE plans (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    title       TEXT NOT NULL,
    plan_type   TEXT NOT NULL DEFAULT 'other' CHECK (plan_type IN ('doctor_visit','take_med','rest','other')),
    time_of_day TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_plans_patient_day ON plans(patient_id, day_of_week);
