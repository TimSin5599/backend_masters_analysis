-- Add record_type to work experience entries
-- This allows categorizing professional development into internships, work history, or training
-- The document type is inferred from the record_type of entries associated with the document

ALTER TABLE applicants_data_work_experience
ADD COLUMN IF NOT EXISTS record_type VARCHAR(255);
