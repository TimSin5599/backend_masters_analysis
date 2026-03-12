-- Add semester GPA fields to transcripts
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS gpa_semester_1 DECIMAL(5,2);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS gpa_semester_2 DECIMAL(5,2);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS gpa_semester_3 DECIMAL(5,2);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS gpa_semester_4 DECIMAL(5,2);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS gpa_semester_5 DECIMAL(5,2);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS gpa_semester_6 DECIMAL(5,2);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS gpa_semester_7 DECIMAL(5,2);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS gpa_semester_8 DECIMAL(5,2);
