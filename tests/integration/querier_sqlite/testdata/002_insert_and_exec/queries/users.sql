-- piko.query(name: InsertUser, command: exec)
INSERT INTO users (id, name, email) VALUES (?, ?, ?);

-- piko.query(name: GetUser, command: one)
SELECT id, name, email FROM users WHERE id = ?;
