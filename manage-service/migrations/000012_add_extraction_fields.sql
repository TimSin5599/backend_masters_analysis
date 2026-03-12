-- Add fields to language training
ALTER TABLE applicants_data_language_training
ADD COLUMN IF NOT EXISTS exam_name VARCHAR(100),
ADD COLUMN IF NOT EXISTS score VARCHAR(50);

-- Add fields to achievements
ALTER TABLE applicants_data_achievements
ADD COLUMN IF NOT EXISTS program VARCHAR(255),
ADD COLUMN IF NOT EXISTS company VARCHAR(255),
ADD COLUMN IF NOT EXISTS end_date DATE;

-- Add fields to motivation
ALTER TABLE applicants_data_motivation
ADD COLUMN IF NOT EXISTS main_text TEXT;
