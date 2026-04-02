-- piko.name: GetUser
-- piko.command: one
SELECT id, name, email, created_at FROM users WHERE id = ?;

-- piko.name: ListUsers
-- piko.command: many
SELECT id, name, email FROM users ORDER BY name;
