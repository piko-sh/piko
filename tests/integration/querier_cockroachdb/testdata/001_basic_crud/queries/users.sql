-- piko.name: GetUser
-- piko.command: one
SELECT id, name, email FROM users WHERE id = $1;

-- piko.name: ListUsers
-- piko.command: many
SELECT id, name, email FROM users ORDER BY id;

-- piko.name: InsertUser
-- piko.command: one
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email;
