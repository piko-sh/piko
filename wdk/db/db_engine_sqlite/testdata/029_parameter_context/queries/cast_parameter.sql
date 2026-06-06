-- piko.query(name: CastParam, command: many)
SELECT id, name FROM products WHERE id = CAST(? AS INTEGER)
