CREATE TABLE IF NOT EXISTS division_field_rules (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    division_id UUID        NOT NULL REFERENCES divisions(id) ON DELETE CASCADE,
    field_id    TEXT        NOT NULL,
    rule_type   TEXT        NOT NULL CHECK (rule_type IN ('allowed', 'preferred')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(division_id, field_id, rule_type)
);
