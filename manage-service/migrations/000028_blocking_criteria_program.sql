-- 000028_blocking_criteria_program.sql

-- 1. Rename ALTERNATIVE → BLOCKING
-- Drop old CHECK constraint and add new one
ALTER TABLE evaluation_criteria
    DROP CONSTRAINT IF EXISTS evaluation_criteria_type_check;

ALTER TABLE evaluation_criteria
    ADD CONSTRAINT evaluation_criteria_type_check
        CHECK (type IN ('BASE', 'BLOCKING'));

UPDATE evaluation_criteria SET type = 'BLOCKING' WHERE type = 'ALTERNATIVE';

-- 2. Add program_id to evaluation_criteria (NULL = applies to all programs)
ALTER TABLE evaluation_criteria
    ADD COLUMN IF NOT EXISTS program_id BIGINT REFERENCES programs(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS criteria_program_idx ON evaluation_criteria (program_id);
