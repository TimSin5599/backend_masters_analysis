-- Add yearly GPA fields to transcripts
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS gpa_year_1 DECIMAL(5,2);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS gpa_year_2 DECIMAL(5,2);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS gpa_year_3 DECIMAL(5,2);
ALTER TABLE applicants_data_transcript ADD COLUMN IF NOT EXISTS gpa_year_4 DECIMAL(5,2);

-- Table for detailed grades
CREATE TABLE IF NOT EXISTS "applicants_data_academic_records" (
    "id" BIGSERIAL PRIMARY KEY,
    "applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
    "document_id" BIGINT REFERENCES "applicants_document"("id"),
    "semester" VARCHAR(50),
    "subject_name" VARCHAR(255) NOT NULL,
    "grade" VARCHAR(50),
    "credits" DECIMAL(10,2),
    "source" VARCHAR(50) NOT NULL, -- 'model' or 'operator'
    "version" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX "academic_records_applicant_idx" ON "applicants_data_academic_records" ("applicant_id");
