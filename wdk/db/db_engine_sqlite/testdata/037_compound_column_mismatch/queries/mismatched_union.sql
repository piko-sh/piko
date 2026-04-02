-- piko.name: GetMismatchedUnion
-- piko.command: many
SELECT id, name FROM users
UNION
SELECT id, name, email FROM users
