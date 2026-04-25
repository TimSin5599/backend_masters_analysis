-- Add document_id to all data tables to link records to source documents

ALTER TABLE "applicants_data_identification" ADD COLUMN IF NOT EXISTS "document_id" BIGINT REFERENCES "applicants_document"("id");
ALTER TABLE "applicants_data_education" ADD COLUMN IF NOT EXISTS "document_id" BIGINT REFERENCES "applicants_document"("id");
ALTER TABLE "applicants_data_transcript" ADD COLUMN IF NOT EXISTS "document_id" BIGINT REFERENCES "applicants_document"("id");
ALTER TABLE "applicants_data_work_experience" ADD COLUMN IF NOT EXISTS "document_id" BIGINT REFERENCES "applicants_document"("id");
ALTER TABLE "applicants_data_language_training" ADD COLUMN IF NOT EXISTS "document_id" BIGINT REFERENCES "applicants_document"("id");
ALTER TABLE "applicants_data_motivation" ADD COLUMN IF NOT EXISTS "document_id" BIGINT REFERENCES "applicants_document"("id");
ALTER TABLE "applicants_data_recommendation" ADD COLUMN IF NOT EXISTS "document_id" BIGINT REFERENCES "applicants_document"("id");
ALTER TABLE "applicants_data_achievements" ADD COLUMN IF NOT EXISTS "document_id" BIGINT REFERENCES "applicants_document"("id");
ALTER TABLE "applicants_data_resume" ADD COLUMN IF NOT EXISTS "document_id" BIGINT REFERENCES "applicants_document"("id");
