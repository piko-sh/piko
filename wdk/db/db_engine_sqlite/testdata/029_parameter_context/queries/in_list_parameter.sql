-- piko.name: InListParam
-- piko.command: many
SELECT id, name FROM products WHERE id IN (?, ?, ?)
