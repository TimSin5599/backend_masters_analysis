-- 000016_expert_system_upgrades.sql

-- 1. Create Expert Slots table for global assignment
CREATE TABLE IF NOT EXISTS "expert_slots" (
    "user_id" BIGINT PRIMARY KEY,
    "slot_number" INTEGER NOT NULL CHECK(slot_number BETWEEN 1 AND 3) UNIQUE,
    "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 2. Update expert_evaluations table
-- First, drop existing table if it's empty or handle migration of data if needed. 
-- Since we are in development and the structure is significantly different, we'll refine it.
DROP TABLE IF EXISTS "expert_evaluations";

CREATE TABLE "expert_evaluations" (
    "id" BIGSERIAL PRIMARY KEY,
    "applicant_id" BIGINT NOT NULL REFERENCES "applicants"("id") ON DELETE CASCADE,
    "expert_id" BIGINT NOT NULL,
    "category" VARCHAR(50) NOT NULL,
    "score" INTEGER NOT NULL CHECK(score >= 0),
    "comment" TEXT,
    "updated_by_id" BIGINT,
    "is_admin_override" BOOLEAN NOT NULL DEFAULT FALSE,
    "source_info" VARCHAR(255),
    "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE("applicant_id", "expert_id", "category")
);

CREATE INDEX "expert_eval_applicant_idx" ON "expert_evaluations" ("applicant_id");
