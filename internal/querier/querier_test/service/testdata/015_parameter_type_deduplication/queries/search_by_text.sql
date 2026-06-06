-- piko.query(name: SearchByText, command: many)
SELECT id, name, email FROM users WHERE name = $1 OR email = $1
