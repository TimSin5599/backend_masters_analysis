-- Simplify structure: remove educational_program table
-- 1. Drop foreign key constraint from applicants
ALTER TABLE applicants DROP CONSTRAINT IF EXISTS applicants_program_id_fkey;

-- 2. Drop the program_id column or rename it (keeping as BIGINT for now if needed, but we will hardcode logic)
-- Actually let's keep the column but it won't reference anything.
-- ALTER TABLE applicants DROP COLUMN IF EXISTS program_id;

-- 3. Drop the educational_program table (renamed to programs in some versions)
DROP TABLE IF EXISTS educational_program CASCADE;
DROP TABLE IF EXISTS programs CASCADE;

-- Note: We'll hardcode "Системная и программная инженерия" in the UI and backend logic.
