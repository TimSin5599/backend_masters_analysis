ALTER TABLE applicants_document ADD COLUMN IF NOT EXISTS processed_at TIMESTAMPTZ;
