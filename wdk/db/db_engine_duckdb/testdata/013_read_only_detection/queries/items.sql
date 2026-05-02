-- piko.query(name: SelectItems, command: many)
SELECT id, name FROM items;

-- piko.query(name: InsertItem, command: exec)
INSERT INTO items (id, name) VALUES ($1, $2);

-- piko.query(name: UpdateItem, command: exec)
UPDATE items SET name = $2 WHERE id = $1;

-- piko.query(name: DeleteItem, command: exec)
DELETE FROM items WHERE id = $1;
