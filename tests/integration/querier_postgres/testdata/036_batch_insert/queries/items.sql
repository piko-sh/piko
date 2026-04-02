-- piko.name: InsertItemsBatch
-- piko.command: batch
INSERT INTO items (id, name, category, price, description) VALUES ($1, $2, $3, $4, $5);

-- piko.name: CountItems
-- piko.command: one
SELECT COUNT(*) AS total FROM items;

-- piko.name: ListItems
-- piko.command: many
SELECT id, name, category, price, description FROM items ORDER BY id ASC;

-- piko.name: GetItem
-- piko.command: one
SELECT id, name, category, price, description FROM items WHERE id = $1;
