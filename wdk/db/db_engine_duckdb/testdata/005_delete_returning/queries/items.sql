-- piko.query(name: DeleteItem, command: one)
DELETE FROM items WHERE id = $1 RETURNING id, name;
