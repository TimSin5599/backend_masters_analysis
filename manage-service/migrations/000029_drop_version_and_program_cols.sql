-- 000029_drop_version_and_program_cols.sql

-- Remove version column from all applicants_data_* tables
ALTER TABLE applicants_data_identification    DROP COLUMN IF EXISTS version;
ALTER TABLE applicants_data_education         DROP COLUMN IF EXISTS version;
ALTER TABLE applicants_data_transcript        DROP COLUMN IF EXISTS version;
ALTER TABLE applicants_data_work_experience   DROP COLUMN IF EXISTS version;
ALTER TABLE applicants_data_language_training DROP COLUMN IF EXISTS version;
ALTER TABLE applicants_data_motivation        DROP COLUMN IF EXISTS version;
ALTER TABLE applicants_data_recommendation    DROP COLUMN IF EXISTS version;
ALTER TABLE applicants_data_achievements      DROP COLUMN IF EXISTS version;
ALTER TABLE applicants_data_resume            DROP COLUMN IF EXISTS version;
ALTER TABLE applicants_data_video             DROP COLUMN IF EXISTS version;

-- Remove program (work context field) column from achievements
ALTER TABLE applicants_data_achievements DROP COLUMN IF EXISTS program;
