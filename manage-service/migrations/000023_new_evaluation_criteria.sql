-- 000023_new_evaluation_criteria.sql

-- Clear old criteria to avoid confusion (since this is a major update to the scoring system)
DELETE FROM "evaluation_criteria";

-- Standard Scheme Criteria
INSERT INTO "evaluation_criteria" (code, title, max_score, type) VALUES
('EDU_BASE', 'Базовое образование', 20, 'BASE'),
('EDU_ADD', 'Дополнительное образование', 25, 'BASE'),
('ACHIEVEMENTS', 'Личные достижения', 30, 'BASE'),
('VIDEO', 'Видео-самопрезентация', 14, 'BASE'),
('MOTIVATION', 'Мотивационное письмо', 5, 'BASE'),
('RECOMMENDATION', 'Рекомендательные письма', 5, 'BASE'),
('ENGLISH', 'Результат по английскому языку', 1, 'ALTERNATIVE');

-- IEEE / International Scheme Criteria (Some are shared, but these are specific to the trigger)
INSERT INTO "evaluation_criteria" (code, title, max_score, type) VALUES
('IEEE_INT', 'Сертификат IEEE / Международный финал', 40, 'BASE'),
('ADD_ACHIEV_COMBINED', 'Доп. образование и достижения', 25, 'BASE')
ON CONFLICT (code) DO NOTHING;

-- Note: In GetEvaluationCriteria usecase, we will filter which ones to return.
