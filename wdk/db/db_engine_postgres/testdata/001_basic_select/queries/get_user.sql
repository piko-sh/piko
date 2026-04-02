-- piko.name: GetUser
-- piko.command: one
SELECT id, name, email, age, created_at FROM users WHERE id = $1;
