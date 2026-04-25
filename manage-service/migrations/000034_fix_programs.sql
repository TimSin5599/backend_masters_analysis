-- Remove test program for 2025 (AI & Robotics)
DELETE FROM programs WHERE title = 'AI & Robotics' AND year = 2025;

-- Rename remaining programs to the real name with correct years
UPDATE programs SET title = 'Системная и программная инженерия', year = 2024
WHERE title = 'Master of Data Science';

UPDATE programs SET title = 'Системная и программная инженерия', year = 2025
WHERE title = 'Software Engineering';
