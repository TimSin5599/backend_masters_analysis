-- Add cumulative fields
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS cumulative_gpa DECIMAL(5,2);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS cumulative_grade VARCHAR(50);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS obtained_credits DECIMAL(10,2);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS total_semesters INTEGER;
