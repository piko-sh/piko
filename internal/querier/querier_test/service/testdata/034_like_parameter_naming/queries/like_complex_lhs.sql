-- piko.name: SearchByConcat
-- piko.command: many
SELECT role FROM users WHERE (name || ' ' || role) LIKE $1;
