-- Originally intended to remove programs table, but it was restored in later migrations.
-- Only drop the old educational_program alias if it exists; keep programs table.
ALTER TABLE applicants DROP CONSTRAINT IF EXISTS applicants_program_id_fkey;

DROP TABLE IF EXISTS educational_program CASCADE;
