-- piko.name: SearchNameCollision
-- piko.command: many
SELECT id FROM users WHERE name LIKE $1 OR name LIKE $2;
