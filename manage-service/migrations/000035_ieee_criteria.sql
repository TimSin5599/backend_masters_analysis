-- Add IEEE-specific evaluation criteria.
-- program_id = NULL means criteria apply to all programs.

INSERT INTO evaluation_criteria (code, title, max_score, type, document_types, is_mandatory, scheme)
VALUES
    ('IEEE_ENGLISH',   'Результат по английскому языку', 1,  'BLOCKING', ARRAY['language_training'], false, 'ieee'),
    ('IEEE_VIDEO',     'Видео самопредставление',        15, 'BASE',     ARRAY['video_presentation'], true,  'ieee'),
    ('IEEE_EDU_BASE',  'Базовое образование',            20, 'BASE',     ARRAY['diploma', 'transcript', 'resume'], true, 'ieee')
ON CONFLICT (code) DO NOTHING;
