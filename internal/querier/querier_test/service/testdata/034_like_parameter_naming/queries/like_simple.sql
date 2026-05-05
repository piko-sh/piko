-- piko.name: SearchByName
-- piko.command: many
SELECT email FROM users WHERE name LIKE $1;
