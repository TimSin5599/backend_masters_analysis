-- Migration 000031: Add scoring scheme override per applicant
-- Values: 'auto' (detect from IEEE/international documents), 'default', 'ieee'
-- 'auto' means the system detects the scheme automatically from achievements.

ALTER TABLE applicants
    ADD COLUMN IF NOT EXISTS scoring_scheme VARCHAR(20) NOT NULL DEFAULT 'auto';
