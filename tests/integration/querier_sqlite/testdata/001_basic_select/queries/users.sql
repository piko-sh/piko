-- piko.query(name: GetUser, command: one)
SELECT id, name, email FROM users WHERE id = ?;

-- piko.query(name: ListUsers, command: many)
SELECT id, name, email FROM users ORDER BY id;
