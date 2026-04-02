-- piko.name: DeleteItem
-- piko.command: one
DELETE FROM items WHERE id = $1 RETURNING id, name;
