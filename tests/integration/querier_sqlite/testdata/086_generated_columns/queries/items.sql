-- piko.name: InsertItem
-- piko.command: exec
INSERT INTO items (id, price, quantity) VALUES (?, ?, ?);

-- piko.name: GetItem
-- piko.command: one
SELECT id, price, quantity, total, label FROM items WHERE id = ?;

-- piko.name: ListByMinTotal
-- piko.command: many
SELECT id, price, quantity, total FROM items WHERE total >= ? ORDER BY total ASC;
