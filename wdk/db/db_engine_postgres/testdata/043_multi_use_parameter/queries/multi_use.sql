-- piko.name: MultiUse
-- piko.command: many
SELECT id, name FROM users WHERE id = $1 OR name = $1::text;
