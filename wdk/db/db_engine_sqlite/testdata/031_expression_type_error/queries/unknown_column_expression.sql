-- piko.query(name: GetComputed, command: many)
SELECT id, missing_column + 1 AS computed FROM users WHERE id = ?
