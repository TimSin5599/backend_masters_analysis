-- Migration: Drop detailed transcript tables and columns
-- Part of simplifying transcript storage

-- 1. Drop detailed tables
DROP TABLE IF EXISTS "applicants_data_academic_records" CASCADE;
DROP TABLE IF EXISTS "applicants_data_transcript_semesters" CASCADE;

-- 2. Drop unnecessary columns from applicants_data_transcript
ALTER TABLE "applicants_data_transcript" 
    DROP COLUMN IF EXISTS "gpa_year_1",
    DROP COLUMN IF EXISTS "gpa_year_2",
    DROP COLUMN IF EXISTS "gpa_year_3",
    DROP COLUMN IF EXISTS "gpa_year_4",
    DROP COLUMN IF EXISTS "gpa_semester_1",
    DROP COLUMN IF EXISTS "gpa_semester_2",
    DROP COLUMN IF EXISTS "gpa_semester_3",
    DROP COLUMN IF EXISTS "gpa_semester_4",
    DROP COLUMN IF EXISTS "gpa_semester_5",
    DROP COLUMN IF EXISTS "gpa_semester_6",
    DROP COLUMN IF EXISTS "gpa_semester_7",
    DROP COLUMN IF EXISTS "gpa_semester_8",
    DROP COLUMN IF EXISTS "gpa_scale",
    DROP COLUMN IF EXISTS "language_of_instruction";
