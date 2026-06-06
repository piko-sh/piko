-- piko.query(name: GetUser, command: one)
SELECT id, name, email FROM users WHERE id = $1;

-- piko.query(name: ListUsers, command: many)
SELECT id, name, email FROM users ORDER BY id;

-- piko.query(name: InsertUser, command: one)
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email;
