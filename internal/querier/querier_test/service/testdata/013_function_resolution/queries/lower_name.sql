-- piko.name: GetLoweredName
-- piko.command: one
SELECT lower(name) as lowered_name FROM users WHERE id = $1
