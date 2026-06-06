-- piko.query(name: FindCaseInsensitive, command: many)
SELECT id, name FROM users WHERE name COLLATE NOCASE = ?
