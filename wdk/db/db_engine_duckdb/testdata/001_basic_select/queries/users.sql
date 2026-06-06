-- piko.query(name: GetAllUsers, command: many)
SELECT id, name, email, active, created_at FROM users;

-- piko.query(name: GetUserByID, command: one)
SELECT id, name, email, active, created_at FROM users WHERE id = $1;

-- piko.query(name: GetActiveUsers, command: many)
SELECT id, name, email FROM users WHERE active = true;
