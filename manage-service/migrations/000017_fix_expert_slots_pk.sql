-- 000017_fix_expert_slots_pk.sql

-- Drop the old table and recreate it with the correct PK
-- We don't want user_id to be PK because a user might need to be moved between slots
-- or we might want to update a slot with a new user.

DROP TABLE IF EXISTS "expert_slots";

CREATE TABLE "expert_slots" (
    "slot_number" INTEGER PRIMARY KEY CHECK(slot_number BETWEEN 1 AND 3),
    "user_id" BIGINT NOT NULL,
    "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- If we want to ensure a user is only in one slot at a time:
CREATE UNIQUE INDEX "expert_slots_user_id_idx" ON "expert_slots" ("user_id");
