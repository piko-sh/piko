-- piko.name: FindCaseInsensitive
-- piko.command: many
SELECT id, name FROM users WHERE name COLLATE NOCASE = ?
