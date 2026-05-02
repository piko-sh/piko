-- piko.query(name: SearchByName, command: many)
SELECT email FROM users WHERE name LIKE $1;
