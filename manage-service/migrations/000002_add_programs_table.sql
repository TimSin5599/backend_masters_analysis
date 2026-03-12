CREATE TABLE IF NOT EXISTS programs (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    year INTEGER NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Наполнение тестовыми данными
INSERT INTO programs (title, year, description) VALUES 
('Master of Data Science', 2024, 'Программа подготовки специалистов в области анализа данных и машинного обучения.'),
('Software Engineering', 2024, 'Современные методы разработки ПО и управления проектами.'),
('AI & Robotics', 2025, 'Исследования в области искусственного интеллекта и автономных систем.');

-- Добавление внешнего ключа (если еще нет, хотя в 000001 он уже мог быть как число)
-- В 000001 таблица applicants уже имеет program_id
ALTER TABLE applicants ADD CONSTRAINT fk_program FOREIGN KEY (program_id) REFERENCES programs(id);
