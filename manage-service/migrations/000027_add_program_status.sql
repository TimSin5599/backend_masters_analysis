-- 000027_add_program_status.sql

ALTER TABLE programs
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'completed'));
