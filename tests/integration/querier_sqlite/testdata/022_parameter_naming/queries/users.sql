-- piko.name: FindUserByEmail
-- piko.command: one
-- ?1 as piko.param(email)
SELECT id, name, email FROM users WHERE email = ?;

-- piko.name: InsertUser
-- piko.command: exec
-- ?1 as piko.param(name)
-- ?2 as piko.param(email)
INSERT INTO users (name, email) VALUES (?, ?);
