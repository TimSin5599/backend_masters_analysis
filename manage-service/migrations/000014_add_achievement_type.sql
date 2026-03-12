-- Migrate achievements to track type
ALTER TABLE applicants_data_achievements
ADD COLUMN IF NOT EXISTS achievement_type VARCHAR(255);
