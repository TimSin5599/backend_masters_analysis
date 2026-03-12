-- Remove legacy foreign key applicants_program_id_fkey targeting educational_program table
ALTER TABLE applicants DROP CONSTRAINT IF EXISTS applicants_program_id_fkey;
