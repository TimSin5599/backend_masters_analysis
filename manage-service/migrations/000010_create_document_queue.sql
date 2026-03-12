CREATE TABLE document_processing_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    applicant_id BIGINT NOT NULL REFERENCES applicants(id) ON DELETE CASCADE,
    document_category VARCHAR(50) NOT NULL,
    file_path VARCHAR(255) NOT NULL,
    priority INT NOT NULL DEFAULT 5,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_document_queue_status_priority ON document_processing_queue(status, priority, created_at);
