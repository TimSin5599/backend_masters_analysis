-- 000026_criteria_document_types.sql

ALTER TABLE evaluation_criteria
    ADD COLUMN IF NOT EXISTS document_types TEXT[]      NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS is_mandatory   BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS scheme         VARCHAR(20) NOT NULL DEFAULT 'default'
        CHECK (scheme IN ('default', 'ieee'));

-- Default scheme
UPDATE evaluation_criteria SET
    document_types = ARRAY['diploma', 'transcript', 'resume'],
    is_mandatory   = true,
    scheme         = 'default'
WHERE code = 'EDU_BASE';

UPDATE evaluation_criteria SET
    document_types = ARRAY['second_diploma', 'prof_development', 'certification'],
    is_mandatory   = false,
    scheme         = 'default'
WHERE code = 'EDU_ADD';

UPDATE evaluation_criteria SET
    document_types = ARRAY['achievement'],
    is_mandatory   = false,
    scheme         = 'default'
WHERE code = 'ACHIEVEMENTS';

UPDATE evaluation_criteria SET
    document_types = ARRAY['video_presentation'],
    is_mandatory   = true,
    scheme         = 'default'
WHERE code = 'VIDEO';

UPDATE evaluation_criteria SET
    document_types = ARRAY['motivation'],
    is_mandatory   = true,
    scheme         = 'default'
WHERE code = 'MOTIVATION';

UPDATE evaluation_criteria SET
    document_types = ARRAY['recommendation'],
    is_mandatory   = true,
    scheme         = 'default'
WHERE code = 'RECOMMENDATION';

UPDATE evaluation_criteria SET
    document_types = ARRAY['language'],
    is_mandatory   = false,
    scheme         = 'default'
WHERE code = 'ENGLISH';

-- IEEE scheme
UPDATE evaluation_criteria SET
    document_types = ARRAY['achievement', 'certification'],
    is_mandatory   = true,
    scheme         = 'ieee'
WHERE code = 'IEEE_INT';

UPDATE evaluation_criteria SET
    document_types = ARRAY['achievement', 'certification', 'second_diploma', 'prof_development'],
    is_mandatory   = false,
    scheme         = 'ieee'
WHERE code = 'ADD_ACHIEV_COMBINED';
