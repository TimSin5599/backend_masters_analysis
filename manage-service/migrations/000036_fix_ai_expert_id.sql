-- 000036_fix_ai_expert_id.sql
-- Fix: expert_id and updated_by_id were UUID, but AI system stores "AI_SYSTEM" (non-UUID string).
-- Solution: convert to TEXT to support both real UUIDs and the AI_SYSTEM sentinel.
-- Also add is_ai_generated flag so frontend can distinguish AI drafts from human scores.

ALTER TABLE expert_evaluations
    ALTER COLUMN expert_id TYPE TEXT USING expert_id::text,
    ALTER COLUMN updated_by_id TYPE TEXT USING updated_by_id::text;

ALTER TABLE expert_evaluations
    ADD COLUMN IF NOT EXISTS is_ai_generated BOOLEAN NOT NULL DEFAULT FALSE;
