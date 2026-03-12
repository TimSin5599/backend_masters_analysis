-- Migration to add initial admin user
INSERT INTO users (email, password, role) 
VALUES ('admin@flatlogic.com', '$2a$10$YjH0N8Wn0W8p8p/m45/n.eB0wM.BwFq2/EHTmYlA/GvE9w3K9iAhm', 'admin')
ON CONFLICT (email) DO NOTHING;
