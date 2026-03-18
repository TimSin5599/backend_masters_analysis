-- Migration: observer -> expert
UPDATE users SET role = 'expert' WHERE role = 'observer';
