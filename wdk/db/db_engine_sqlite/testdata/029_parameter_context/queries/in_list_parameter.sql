-- piko.query(name: InListParam, command: many)
SELECT id, name FROM products WHERE id IN (?, ?, ?)
