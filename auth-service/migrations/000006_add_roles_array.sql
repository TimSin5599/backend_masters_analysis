-- Migration: role VARCHAR(50) -> roles TEXT[]
-- 1. Add new column
ALTER TABLE users ADD COLUMN IF NOT EXISTS roles TEXT[] NOT NULL DEFAULT '{}';

-- 2. Populate from existing role column: operator -> expert, rest unchanged
UPDATE users SET roles = ARRAY[CASE WHEN role = 'operator' THEN 'expert' ELSE role END];

-- 3. Drop old column
ALTER TABLE users DROP COLUMN IF EXISTS role;
