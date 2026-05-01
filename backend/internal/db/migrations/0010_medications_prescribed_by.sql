ALTER TABLE medications
    ADD COLUMN IF NOT EXISTS prescribed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS prescribed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_meds_prescribed_by ON medications(prescribed_by) WHERE prescribed_by IS NOT NULL;
