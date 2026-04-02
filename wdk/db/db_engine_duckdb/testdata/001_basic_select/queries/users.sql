-- piko.name: GetAllUsers
-- piko.command: many
SELECT id, name, email, active, created_at FROM users;

-- piko.name: GetUserByID
-- piko.command: one
SELECT id, name, email, active, created_at FROM users WHERE id = $1;

-- piko.name: GetActiveUsers
-- piko.command: many
SELECT id, name, email FROM users WHERE active = true;
