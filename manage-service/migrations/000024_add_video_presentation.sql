-- 000024_add_video_presentation.up.sql
CREATE TABLE IF NOT EXISTS applicants_data_video (
    id BIGSERIAL PRIMARY KEY,
    applicant_id BIGINT NOT NULL REFERENCES applicants(id) ON DELETE CASCADE,
    video_url TEXT,
    source VARCHAR(50) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Down
-- DROP TABLE IF EXISTS applicants_data_video;
