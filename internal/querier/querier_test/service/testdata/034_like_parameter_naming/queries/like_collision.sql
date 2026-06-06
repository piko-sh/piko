-- piko.query(name: SearchNameCollision, command: many)
SELECT id FROM users WHERE name LIKE $1 OR name LIKE $2;
