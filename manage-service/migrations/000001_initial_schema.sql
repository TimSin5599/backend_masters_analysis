-- Final Integrated Schema for Manage Service (Version 3)

-- 1. Educational Programs
CREATE TABLE IF NOT EXISTS "educational_program" (
	"id" BIGSERIAL PRIMARY KEY,
	"name" VARCHAR(255) NOT NULL,
	"year" INTEGER NOT NULL,
	"description" TEXT,
	"status" VARCHAR(255) NOT NULL,
	"created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Applicants
CREATE TABLE IF NOT EXISTS "applicants" (
	"id" BIGSERIAL PRIMARY KEY,
	"program_id" BIGINT REFERENCES "educational_program"("id"),
	"status" VARCHAR(50) NOT NULL DEFAULT 'uploaded', -- uploaded, processed, verified, evaluated
	"created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"updated_at" TIMESTAMP
);
CREATE INDEX "applicants_index_0" ON "applicants" ("program_id", "status");

-- 3. Documents
CREATE TABLE IF NOT EXISTS "applicants_document" (
	"id" BIGSERIAL PRIMARY KEY,
	"applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
	"file_type" VARCHAR(255) NOT NULL,
	"file_name" VARCHAR(255) NOT NULL,
	"storage_path" TEXT NOT NULL,
	"uploaded_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX "applicants_document_index_0" ON "applicants_document" ("applicant_id");

-- 4. Extracted Fields Log (AI Drafts Buffer)
CREATE TABLE IF NOT EXISTS "extracted_fields" (
	"id" BIGSERIAL PRIMARY KEY,
	"applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
	"document_id" BIGINT NOT NULL REFERENCES "applicants_document"("id"),
	"field_name" VARCHAR(255) NOT NULL,
	"field_value" TEXT,
	"extracted_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 5. Typed Data Tables (Versioned)

-- Identification (Bio)
CREATE TABLE IF NOT EXISTS "applicants_data_identification" (
	"id" BIGSERIAL PRIMARY KEY,
	"applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
	"email" VARCHAR(255),
	"phone" VARCHAR(255),
	"document_number" VARCHAR(255),
	"name" VARCHAR(255),
	"surname" VARCHAR(255),
	"patronymic" VARCHAR(255),
	"date_of_birth" DATE,
	"gender" VARCHAR(50),
	"nationality" VARCHAR(255),
	"photo_path" TEXT, -- Path to applicant's photo in MinIO
	"source" VARCHAR(50) NOT NULL, -- 'model' or 'operator'
    "version" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Education (Bachelor Diploma)
CREATE TABLE IF NOT EXISTS "applicants_data_education" (
    "id" BIGSERIAL PRIMARY KEY,
    "applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
    "institution_name" VARCHAR(255),
    "degree_title" VARCHAR(255),
    "major" VARCHAR(255),
    "graduation_date" DATE,
    "diploma_serial_number" VARCHAR(255),
    "source" VARCHAR(50) NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Transcript (Grades/GPA)
CREATE TABLE IF NOT EXISTS "applicants_data_transcript" (
    "id" BIGSERIAL PRIMARY KEY,
    "applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
    "gpa" DECIMAL(5,2),
    "gpa_scale" VARCHAR(20),
    "total_credits" DECIMAL(10,2),
    "language_of_instruction" VARCHAR(50),
    "source" VARCHAR(50) NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Work Experience
CREATE TABLE IF NOT EXISTS "applicants_data_work_experience" (
	"id" BIGSERIAL PRIMARY KEY,
	"applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
	"country" VARCHAR(100),
	"city" VARCHAR(100),
	"position" VARCHAR(255),
	"company_name" VARCHAR(255),
	"start_date" DATE,
	"end_date" DATE,
	"source" VARCHAR(50) NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Language Training & Certificates
CREATE TABLE IF NOT EXISTS "applicants_data_language_training" (
	"id" BIGSERIAL PRIMARY KEY,
	"applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
	"russian_level" VARCHAR(50),
	"english_level" VARCHAR(50),
    "certificate_path" TEXT, -- Path to language cert (TOEFL/IELTS etc)
    "source" VARCHAR(50) NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Motivation Analysis
CREATE TABLE IF NOT EXISTS "applicants_data_motivation" (
    "id" BIGSERIAL PRIMARY KEY,
    "applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
    "reasons_for_applying" TEXT,
    "experience_summary" TEXT,
    "career_goals" TEXT,
    "detected_language" VARCHAR(50),
    "source" VARCHAR(50) NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Recommendation Letters
CREATE TABLE IF NOT EXISTS "applicants_data_recommendation" (
    "id" BIGSERIAL PRIMARY KEY,
    "applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
    "author_name" VARCHAR(255),
    "author_position" VARCHAR(255),
    "author_institution" VARCHAR(255),
    "key_strengths" TEXT,
    "source" VARCHAR(50) NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Achievements
CREATE TABLE IF NOT EXISTS "applicants_data_achievements" (
    "id" BIGSERIAL PRIMARY KEY,
    "applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
    "achievement_title" VARCHAR(255),
    "description" TEXT,
    "date_received" DATE,
    "document_path" TEXT, -- Path to proof document
    "source" VARCHAR(50) NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Resume / Questionnaire Summary
CREATE TABLE IF NOT EXISTS "applicants_data_resume" (
    "id" BIGSERIAL PRIMARY KEY,
    "applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
    "summary" TEXT,
    "skills" TEXT[], -- Array of skills
    "source" VARCHAR(50) NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 6. Expert Evaluations
CREATE TABLE IF NOT EXISTS "expert_evaluations" (
	"id" BIGSERIAL PRIMARY KEY,
	"applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
	"expert_id" BIGINT, -- Links to users.id in auth-service
	"score" INTEGER NOT NULL CHECK(score >= 0),
	"comment" TEXT,
	"created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX "expert_evaluations_index_0" ON "expert_evaluations" ("score");

-- 7. Operator Actions (Audit Log)
CREATE TABLE IF NOT EXISTS "operator_actions" (
	"id" BIGSERIAL PRIMARY KEY,
	"applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id"),
	"operator_id" BIGINT NOT NULL,
	"action_type" VARCHAR(100),
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
