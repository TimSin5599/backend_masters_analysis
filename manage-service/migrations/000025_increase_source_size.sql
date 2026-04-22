-- 000025_increase_source_size.sql

ALTER TABLE applicants_data_identification ALTER COLUMN source TYPE VARCHAR(255);
ALTER TABLE applicants_data_language_training ALTER COLUMN source TYPE VARCHAR(255);
ALTER TABLE applicants_data_motivation ALTER COLUMN source TYPE VARCHAR(255);
ALTER TABLE applicants_data_recommendation ALTER COLUMN source TYPE VARCHAR(255);
ALTER TABLE applicants_data_resume ALTER COLUMN source TYPE VARCHAR(255);
ALTER TABLE applicants_data_transcript ALTER COLUMN source TYPE VARCHAR(255);
ALTER TABLE applicants_data_video ALTER COLUMN source TYPE VARCHAR(255);
ALTER TABLE applicants_data_work_experience ALTER COLUMN source TYPE VARCHAR(255);
ALTER TABLE applicants_data_education ALTER COLUMN source TYPE VARCHAR(255);
