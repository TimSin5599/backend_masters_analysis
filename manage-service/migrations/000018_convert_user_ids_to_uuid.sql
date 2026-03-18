-- 000018_convert_user_ids_to_uuid.sql

-- 1. Исправляем таблицу expert_slots
ALTER TABLE "expert_slots" DROP CONSTRAINT IF EXISTS "expert_slots_user_id_idx";
DROP INDEX IF EXISTS "expert_slots_user_id_idx";

-- Очищаем таблицу, так как старые данные (обрезанные ID) все равно невалидны
TRUNCATE TABLE "expert_slots";

ALTER TABLE "expert_slots" 
    ALTER COLUMN "user_id" TYPE UUID USING NULL; -- Переводим в UUID (данные сбрасываем)

CREATE UNIQUE INDEX "expert_slots_user_id_idx" ON "expert_slots" ("user_id");

-- 2. Исправляем таблицу expert_evaluations
-- Очищаем оценки, так как они привязаны к неверным ID экспертов
TRUNCATE TABLE "expert_evaluations";

ALTER TABLE "expert_evaluations" 
    ALTER COLUMN "expert_id" TYPE UUID USING NULL,
    ALTER COLUMN "updated_by_id" TYPE UUID USING NULL;
