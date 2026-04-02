-- piko.name: SearchUsers
-- piko.command: many
-- $1 as piko.sortable(orderBy) columns:name,email,nonexistent
SELECT id, name, email FROM users WHERE active = true ORDER BY $1
