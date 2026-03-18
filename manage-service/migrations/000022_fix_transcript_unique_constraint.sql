-- Migration: Fix unique constraint on applicants_data_transcript
-- Swap uniqueness from applicant_id to document_id

-- 1. Remove duplicates by document_id before adding constraint (just in case)
DELETE FROM applicants_data_transcript a
USING applicants_data_transcript b
WHERE a.id < b.id 
  AND a.document_id = b.document_id;

-- 2. Drop the old incorrect constraint
ALTER TABLE applicants_data_transcript 
DROP CONSTRAINT IF EXISTS unique_applicant_transcript;

-- 3. Add the correct unique constraint on document_id
ALTER TABLE applicants_data_transcript 
DROP CONSTRAINT IF EXISTS unique_document_transcript;

ALTER TABLE applicants_data_transcript 
ADD CONSTRAINT unique_document_transcript UNIQUE (document_id);
