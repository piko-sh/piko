-- piko.name: FuncArgParam
-- piko.command: many
SELECT id, name FROM products WHERE name = substr(?, 1, ?)
