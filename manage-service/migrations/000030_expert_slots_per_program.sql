-- Migration 000030: Expert slots per program
-- Replaces the global 3-slot system with per-program expert slots.

-- Step 1: Add program_id column as nullable first (avoids FK issues on existing rows)
ALTER TABLE expert_slots ADD COLUMN IF NOT EXISTS program_id bigint;

-- Step 2: Delete all existing global slot assignments (they will be re-assigned per program)
DELETE FROM expert_slots WHERE program_id IS NULL;

-- Step 3: Make NOT NULL and add FK
ALTER TABLE expert_slots ALTER COLUMN program_id SET NOT NULL;

ALTER TABLE expert_slots DROP CONSTRAINT IF EXISTS expert_slots_program_id_fkey;
ALTER TABLE expert_slots ADD CONSTRAINT expert_slots_program_id_fkey
    FOREIGN KEY (program_id) REFERENCES programs(id);

-- Step 4: Drop old PK (slot_number alone) and old unique index
ALTER TABLE expert_slots DROP CONSTRAINT IF EXISTS expert_slots_pkey;
DROP INDEX IF EXISTS expert_slots_user_id_idx;

-- Step 5: New composite PK (slot_number, program_id)
ALTER TABLE expert_slots ADD PRIMARY KEY (slot_number, program_id);

-- Step 6: Drop old slot_number_check if exists, then re-add
ALTER TABLE expert_slots DROP CONSTRAINT IF EXISTS expert_slots_slot_number_check;
ALTER TABLE expert_slots ADD CONSTRAINT expert_slots_slot_number_check
    CHECK (slot_number >= 1 AND slot_number <= 3);

-- Step 7: A user can only occupy one slot per program
DROP INDEX IF EXISTS expert_slots_user_program_idx;
CREATE UNIQUE INDEX expert_slots_user_program_idx ON expert_slots(user_id, program_id);
