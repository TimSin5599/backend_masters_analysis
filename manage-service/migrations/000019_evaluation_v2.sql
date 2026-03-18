-- 000019_evaluation_v2.sql

-- 1. Добавляем колонку статуса в expert_evaluations
ALTER TABLE "expert_evaluations" ADD COLUMN IF NOT EXISTS "status" VARCHAR(20) NOT NULL DEFAULT 'DRAFT';

-- 2. Создаем таблицу критериев
CREATE TABLE IF NOT EXISTS "evaluation_criteria" (
    "code" VARCHAR(50) PRIMARY KEY,
    "title" VARCHAR(255) NOT NULL,
    "max_score" INTEGER NOT NULL CHECK(max_score > 0),
    "type" VARCHAR(20) NOT NULL DEFAULT 'BASE' CHECK(type IN ('BASE', 'ALTERNATIVE')),
    "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. Наполняем базовыми критериями (пример)
INSERT INTO "evaluation_criteria" (code, title, max_score, type) VALUES
('GPA', 'Средний балл диплома', 10, 'BASE'),
('WORK_EXP', 'Опыт работы', 20, 'BASE'),
('MOTIVATION', 'Мотивационное письмо', 15, 'BASE'),
('ENGLISH_LANG', 'Уровень английского языка', 10, 'ALTERNATIVE')
ON CONFLICT (code) DO NOTHING;

-- 4. Добавляем поля для агрегированных результатов в applicants (или можно отдельную таблицу, но для рейтинга быстрее в основной)
ALTER TABLE "applicants" ADD COLUMN IF NOT EXISTS "aggregated_score" DECIMAL(5,2) DEFAULT 0;
ALTER TABLE "applicants" ADD COLUMN IF NOT EXISTS "evaluation_status" VARCHAR(20) DEFAULT 'PENDING';

CREATE INDEX IF NOT EXISTS "applicants_score_idx" ON "applicants" ("aggregated_score" DESC);
