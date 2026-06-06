-- piko.query(name: GetUser, command: one)
SELECT id, name, email FROM users WHERE id = ?;

-- piko.query(name: ListUsers, command: many)
SELECT id, name, email FROM users ORDER BY id;

-- piko.query(name: CreateUser, command: one)
INSERT INTO users (name, email) VALUES (?, ?) RETURNING id, name, email;
