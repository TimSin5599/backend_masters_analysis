-- Migration to add initial admin user
INSERT INTO users (email, password, role) 
VALUES ('admin@masters.com', '$2a$10$aUrOVtWUX2sxq593sHw3/uMFwZWWmUeYQmSs1UDqRhmbEb1wRqVBu', 'admin')
ON CONFLICT (email) DO NOTHING;
