-- piko.name: GetComputed
-- piko.command: many
SELECT id, missing_column + 1 AS computed FROM users WHERE id = ?
