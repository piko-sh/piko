-- piko.query(name: FuncArgParam, command: many)
SELECT id, name FROM products WHERE name = substr(?, 1, ?)
