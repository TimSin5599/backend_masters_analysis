-- 1. Remove duplicates, keeping only the latest one per document
DELETE FROM applicants_data_transcript a
USING applicants_data_transcript b
WHERE a.id < b.id 
  AND a.applicant_id = b.applicant_id;

-- 2. Add unique constraint
ALTER TABLE applicants_data_transcript 
ADD CONSTRAINT unique_applicant_transcript UNIQUE (applicant_id);
