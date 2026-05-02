-- piko.query(name: InsertItemsBatch, command: batch)
INSERT INTO items (id, name, category, price, description) VALUES ($1, $2, $3, $4, $5);

-- piko.query(name: CountItems, command: one)
SELECT COUNT(*) AS total FROM items;

-- piko.query(name: ListItems, command: many)
SELECT id, name, category, price, description FROM items ORDER BY id ASC;

-- piko.query(name: GetItem, command: one)
SELECT id, name, category, price, description FROM items WHERE id = $1;
