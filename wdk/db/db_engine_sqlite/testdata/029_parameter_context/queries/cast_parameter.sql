-- piko.name: CastParam
-- piko.command: many
SELECT id, name FROM products WHERE id = CAST(? AS INTEGER)
