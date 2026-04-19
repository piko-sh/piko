-- piko.query(name: InsertUserReturning, command: one, optional: true)
-- ?1 as piko.param(email)
INSERT INTO users (email) VALUES (?1)
ON CONFLICT (email) DO NOTHING
RETURNING id, email;
