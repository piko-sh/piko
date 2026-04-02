-- piko.name: InsertUser
-- piko.command: exec
INSERT INTO users (id, name, email) VALUES (?, ?, ?);

-- piko.name: GetUser
-- piko.command: one
SELECT id, name, email FROM users WHERE id = ?;
