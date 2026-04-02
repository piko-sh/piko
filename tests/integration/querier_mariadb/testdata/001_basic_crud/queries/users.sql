-- piko.name: GetUser
-- piko.command: one
SELECT id, name, email FROM users WHERE id = ?;

-- piko.name: ListUsers
-- piko.command: many
SELECT id, name, email FROM users ORDER BY id;

-- piko.name: CreateUser
-- piko.command: one
INSERT INTO users (name, email) VALUES (?, ?) RETURNING id, name, email;
