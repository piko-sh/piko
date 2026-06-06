-- piko.query(name: SearchUsers, command: many)
-- piko.sortable(orderBy, columns: [name, email, nonexistent])
SELECT id, name, email FROM users WHERE active = true ORDER BY $1
