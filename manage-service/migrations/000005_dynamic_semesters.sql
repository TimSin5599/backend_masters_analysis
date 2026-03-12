-- Table for dynamic semester GPAs
CREATE TABLE IF NOT EXISTS "applicants_data_transcript_semesters" (
    "id" BIGSERIAL PRIMARY KEY,
    "transcript_id" BIGINT NOT NULL REFERENCES "applicants_data_transcript"("id") ON DELETE CASCADE,
    "semester_number" INTEGER NOT NULL,
    "gpa" DECIMAL(5,2) NOT NULL,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX "transcript_semesters_idx" ON "applicants_data_transcript_semesters" ("transcript_id");
