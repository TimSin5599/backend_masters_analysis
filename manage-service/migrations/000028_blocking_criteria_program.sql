-- 1. Drop old CHECK constraint
ALTER TABLE evaluation_criteria
    DROP CONSTRAINT IF EXISTS evaluation_criteria_type_check;

-- 2. Migrate data BEFORE adding new constraint
UPDATE evaluation_criteria SET type = 'BLOCKING' WHERE type = 'ALTERNATIVE';

-- 3. Add new constraint that allows BLOCKING
ALTER TABLE evaluation_criteria
    ADD CONSTRAINT evaluation_criteria_type_check
        CHECK (type IN ('BASE', 'BLOCKING'));

-- 4. Add program_id to evaluation_criteria (NULL = applies to all programs)
ALTER TABLE evaluation_criteria
    ADD COLUMN IF NOT EXISTS program_id BIGINT REFERENCES programs(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS criteria_program_idx ON evaluation_criteria (program_id);
