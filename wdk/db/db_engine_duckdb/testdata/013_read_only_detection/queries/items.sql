-- piko.name: SelectItems
-- piko.command: many
SELECT id, name FROM items;

-- piko.name: InsertItem
-- piko.command: exec
INSERT INTO items (id, name) VALUES ($1, $2);

-- piko.name: UpdateItem
-- piko.command: exec
UPDATE items SET name = $2 WHERE id = $1;

-- piko.name: DeleteItem
-- piko.command: exec
DELETE FROM items WHERE id = $1;
