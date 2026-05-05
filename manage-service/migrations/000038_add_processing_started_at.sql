ALTER TABLE applicants_document
    ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMPTZ;
