-- Migration to add initial admin user
INSERT INTO users (email, password, role, first_name, last_name) 
VALUES ('admin@masters.com', '$2a$10$aUrOVtWUX2sxq593sHw3/uMFwZWWmUeYQmSs1UDqRhmbEb1wRqVBu', 'admin', 'Admin', 'Admin')
ON CONFLICT (email) DO NOTHING;
