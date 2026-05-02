-- piko.query(name: SearchByConcat, command: many)
SELECT role FROM users WHERE (name || ' ' || role) LIKE $1;
